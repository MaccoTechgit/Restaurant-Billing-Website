package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Where the admin accounts (login e-mail + password hash) live.
// Set MONGO_URI to keep them in MongoDB; otherwise they stay in the local SQLite file.
// The URI is only ever read from the environment (or a git-ignored .env file) - never written into the code.

type adminRec struct {
	Email, Name, Hash string
	Fails             int
	LockedUntil       int64
}

type adminStore interface {
	Get(email string) (*adminRec, error) // nil, nil when there is no such admin
	Count() (int64, error)
	Create(a adminRec) error
	SetFails(email string, fails int, lockedUntil int64) error
	SetHash(email, hash string) error
	Name() string
}

// ---------- SQLite ----------

type sqliteAdmins struct{ db *sql.DB }

func (a *sqliteAdmins) Name() string { return "SQLite" }
func (a *sqliteAdmins) Get(email string) (*adminRec, error) {
	r := &adminRec{}
	err := a.db.QueryRow(`SELECT email,name,hash,fails,locked_until FROM admins WHERE email=?`, email).Scan(&r.Email, &r.Name, &r.Hash, &r.Fails, &r.LockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}
func (a *sqliteAdmins) Count() (int64, error) {
	var n int64
	err := a.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	return n, err
}
func (a *sqliteAdmins) Create(r adminRec) error {
	_, err := a.db.Exec(`INSERT INTO admins(email,hash,name) VALUES(?,?,?)`, r.Email, r.Hash, r.Name)
	return err
}
func (a *sqliteAdmins) SetFails(email string, fails int, lockedUntil int64) error {
	_, err := a.db.Exec(`UPDATE admins SET fails=?, locked_until=? WHERE email=?`, fails, lockedUntil, email)
	return err
}
func (a *sqliteAdmins) SetHash(email, hash string) error {
	_, err := a.db.Exec(`UPDATE admins SET hash=? WHERE email=?`, hash, email)
	return err
}

// ---------- MongoDB ----------

type mongoAdmins struct {
	coll *mongo.Collection
}

type mongoAdminDoc struct {
	Email       string `bson:"email"`
	Name        string `bson:"name"`
	Hash        string `bson:"hash"`
	Fails       int    `bson:"fails"`
	LockedUntil int64  `bson:"locked_until"`
	CreatedAt   int64  `bson:"created_at"`
}

func newMongoAdmins(uri, dbName, collName string) (*mongoAdmins, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetServerSelectionTimeout(8*time.Second).SetConnectTimeout(8*time.Second))
	if err != nil {
		return nil, err
	}
	m := &mongoAdmins{coll: cl.Database(dbName).Collection(collName)}
	return m, nil
}

func (m *mongoAdmins) Name() string { return "MongoDB" }

func (m *mongoAdmins) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 8*time.Second)
}

// ensureIndex makes e-mail unique; safe to call repeatedly.
func (m *mongoAdmins) ensureIndex() error {
	ctx, cancel := m.ctx()
	defer cancel()
	_, err := m.coll.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)})
	return err
}

func (m *mongoAdmins) Get(email string) (*adminRec, error) {
	ctx, cancel := m.ctx()
	defer cancel()
	var d mongoAdminDoc
	err := m.coll.FindOne(ctx, bson.D{{Key: "email", Value: email}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &adminRec{Email: d.Email, Name: d.Name, Hash: d.Hash, Fails: d.Fails, LockedUntil: d.LockedUntil}, nil
}
func (m *mongoAdmins) Count() (int64, error) {
	ctx, cancel := m.ctx()
	defer cancel()
	return m.coll.CountDocuments(ctx, bson.D{})
}
func (m *mongoAdmins) Create(r adminRec) error {
	ctx, cancel := m.ctx()
	defer cancel()
	_, err := m.coll.InsertOne(ctx, mongoAdminDoc{Email: r.Email, Name: r.Name, Hash: r.Hash, CreatedAt: now()})
	return err
}
func (m *mongoAdmins) SetFails(email string, fails int, lockedUntil int64) error {
	ctx, cancel := m.ctx()
	defer cancel()
	_, err := m.coll.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set", Value: bson.D{{Key: "fails", Value: fails}, {Key: "locked_until", Value: lockedUntil}}}})
	return err
}
func (m *mongoAdmins) SetHash(email, hash string) error {
	ctx, cancel := m.ctx()
	defer cancel()
	_, err := m.coll.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set", Value: bson.D{{Key: "hash", Value: hash}}}})
	return err
}

// ---------- selection, seeding, .env ----------

// loadDotEnv reads KEY=VALUE lines from .env (if present) for variables that are not already set.
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func (s *Server) openAdminStore() adminStore {
	uri := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if uri == "" {
		return &sqliteAdmins{db: s.db}
	}
	m, err := newMongoAdmins(uri, env("MONGO_DB", "macbill_site"), env("MONGO_ADMINS", "admins"))
	if err != nil {
		log.Printf("WARNING: could not set up the MongoDB client for admin accounts: %v", err)
		return &unavailableAdmins{err: err}
	}
	return m
}

// unavailableAdmins is used when MONGO_URI is set but invalid: login is refused instead of silently using another store.
type unavailableAdmins struct{ err error }

func (u *unavailableAdmins) Name() string                      { return "MongoDB (unavailable)" }
func (u *unavailableAdmins) Get(string) (*adminRec, error)     { return nil, u.err }
func (u *unavailableAdmins) Count() (int64, error)             { return 0, u.err }
func (u *unavailableAdmins) Create(adminRec) error             { return u.err }
func (u *unavailableAdmins) SetFails(string, int, int64) error { return u.err }
func (u *unavailableAdmins) SetHash(string, string) error      { return u.err }

// seedAdmin creates the first admin when the store is empty. It is retried on login if the database was unreachable at start-up.
func (s *Server) seedAdmin() error {
	if m, ok := s.admins.(*mongoAdmins); ok {
		if err := m.ensureIndex(); err != nil {
			return err
		}
	}
	n, err := s.admins.Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	email, pw := strings.ToLower(env("ADMIN_EMAIL", "admin@macbill.in")), os.Getenv("ADMIN_PASSWORD")
	generated := pw == ""
	if generated {
		b := make([]byte, 6)
		rand.Read(b)
		pw = "Mb-" + hex.EncodeToString(b)
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), 11)
	if err := s.admins.Create(adminRec{Email: email, Name: "Admin", Hash: string(h)}); err != nil {
		return err
	}
	fmt.Printf("\n  First admin login created in %s\n  e-mail: %s\n", s.admins.Name(), email)
	if generated {
		fmt.Printf("  password: %s   (change it after logging in)\n", pw)
	}
	fmt.Println()
	return nil
}
