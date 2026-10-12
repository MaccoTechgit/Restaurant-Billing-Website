package main

// All public-page copy lives here, so a page is added by adding data - not by copying HTML.

type Point struct{ Icon, Title, Text string }
type QA struct{ Q, A string }

// Detail drives both the product pages and the solution pages.
type Detail struct {
	Slug, Kind  string // Kind: product | solution
	Name, Icon  string
	Title, Desc string // <title> and meta description
	H1, Lead    string
	Points      []Point
	Steps       []Point
	Bullets     []string
	FAQ         []QA
	Related     []string // slugs
	BlogLinks   []string // blog slugs
	Short       string   // one line for menus and cards
}

func (d *Detail) Path() string {
	if d.Kind == "solution" {
		return "/solutions/" + d.Slug
	}
	return "/" + d.Slug
}

var products = []*Detail{
	{Slug: "pos-billing", Kind: "product", Name: "POS & Billing", Icon: "🧾",
		Short: "Fast billing for dine-in, takeaway and delivery",
		Title: "Restaurant POS & Billing Software | MacBill System",
		Desc:  "Fast restaurant POS and billing: GST invoices, split bills, UPI/card/cash, 58mm & 80mm receipt printing with barcode and QR. Easy for new staff to learn.",
		H1:    "Restaurant POS & billing that your staff learns in ten minutes",
		Lead:  "Take orders, send them to the kitchen, collect payment and print a clean bill - from one simple screen that works on any PC, laptop or tablet.",
		Points: []Point{
			{"⚡", "Quick order screen", "Big item buttons, categories, search and barcode scan. Add notes, options and quantities in a tap."},
			{"🧮", "GST and charges done right", "Tax, service charge, discounts, vouchers and round-off are calculated for you and shown clearly on the bill."},
			{"💳", "Every way to pay", "Cash, card, UPI, wallet, gift card, loyalty points or on-account - and split one bill across several modes."},
			{"🖨", "Receipts that fit the roll", "Prints exactly on 58mm or 80mm paper - never a full A4 sheet - with your name, cashier, payment mode, barcode and QR."},
			{"👤", "POS operator mode", "A restricted role that can only take orders and print its own bills. Perfect for a busy counter."},
			{"⌨", "Keyboard shortcuts", "Search, send to kitchen, pay and print without touching the mouse, so rush hour moves faster."},
		},
		Steps:   []Point{{"1", "Pick items", "Tap items or scan a barcode. Choose dine-in, takeaway or delivery."}, {"2", "Send to kitchen", "The ticket reaches the kitchen screen instantly - no running back and forth."}, {"3", "Take payment", "Choose the mode, give change, and print or skip the receipt."}},
		Bullets: []string{"Dine-in, takeaway, delivery and QR orders in one place", "Hold, edit and merge bills; move an order between tables", "Customer name on the bill when you want it", "Void and refund with permission control and a full audit trail", "Daily sales, item-wise and payment-mode reports"},
		FAQ: []QA{
			{"Which printers work with it?", "Any normal 58mm or 80mm thermal receipt printer that your computer can print to. There is no special driver or hardware to buy from us."},
			{"Can a waiter print bills?", "Only if you allow it. Roles control this: waiters can take orders and send them to the kitchen, while cashiers and POS operators can print."},
			{"Does it handle GST?", "Yes. You set your tax rates once, and every bill shows the tax. Always confirm the correct rates and rules for your business with your accountant."},
		},
		Related: []string{"kitchen-display", "qr-ordering", "accounting"}, BlogLinks: []string{"how-to-choose-restaurant-billing-software", "gst-billing-for-restaurants-simple-guide"}},

	{Slug: "kitchen-display", Kind: "product", Name: "Kitchen Display", Icon: "👨‍🍳",
		Short: "Orders on a screen, ready alerts for waiters",
		Title: "Kitchen Display System (KDS) | MacBill System",
		Desc:  "Replace paper KOT with a live kitchen screen. New orders appear instantly, chefs mark items ready, and waiters get an automatic alert. Optional auto-print.",
		H1:    "A kitchen screen that ends shouting, lost slips and cold food",
		Lead:  "Every order shows up on the kitchen display the moment it is sent. When the food is ready, the waiter is told automatically - no one has to walk over and ask.",
		Points: []Point{
			{"📺", "Live order board", "New, cooking and ready columns update in real time. The order number is big and clear."},
			{"🔔", "Ready alerts", "When the kitchen taps Ready, the waiter's screen shows a pop-up with the order number and table."},
			{"🖨", "Auto-print if you like", "Keep the screen, or also print a kitchen ticket automatically for each new order. It is a setting you switch on or off."},
			{"🎨", "Colour warnings", "Orders that wait too long change colour, so nothing gets forgotten at a busy time."},
			{"🍳", "Item-level control", "Mark single items done, so the starter can go out while the main course is still cooking."},
			{"🔊", "Sound for new orders", "A short tone tells the kitchen a new ticket has arrived, even if nobody is looking at the screen."},
		},
		Steps:   []Point{{"1", "Order is sent", "The waiter or cashier presses Send to kitchen."}, {"2", "Chef prepares", "The ticket appears on the screen; the chef starts and finishes it."}, {"3", "Waiter is alerted", "A pop-up says the order is ready, so food is served hot."}},
		Bullets: []string{"Works on any tablet, TV with a browser, or a spare monitor", "Separate screens or stations for kitchen, bar and dessert", "Fewer mistakes than handwritten slips", "Less shouting and fewer 'where is my order?' complaints"},
		FAQ: []QA{
			{"Do I need special kitchen hardware?", "No. Any screen with a browser works - a cheap tablet or a wall-mounted monitor is enough."},
			{"Can I still print kitchen tickets?", "Yes. Turn on automatic printing in settings if your kitchen prefers paper, or use screen and paper together."},
			{"What if the kitchen is slow?", "Orders change colour as they wait, and managers can see waiting times in reports."},
		},
		Related: []string{"pos-billing", "qr-ordering", "inventory"}, BlogLinks: []string{"kitchen-display-system-vs-paper-kot"}},

	{Slug: "qr-ordering", Kind: "product", Name: "QR Table Ordering", Icon: "📱",
		Short: "Guests scan, see the menu and order from the table",
		Title: "QR Code Table Ordering | MacBill System",
		Desc:  "Guests scan the QR code on the table, browse your menu and order - no app to install. Branded table cards, orders go straight to the kitchen.",
		H1:    "Let guests scan, browse your menu and order from their table",
		Lead:  "Print a branded QR card for each table. Guests scan it with their phone camera, see your menu with your name and logo, and place an order for that exact table.",
		Points: []Point{
			{"📲", "No app to install", "It opens in the phone's browser. Nothing to download, nothing to sign up for."},
			{"🪑", "Knows the table", "Each card is tied to one table, so the order arrives at the kitchen with the right table name."},
			{"🎨", "Your brand", "The guest page and the printable table card carry your restaurant name and logo."},
			{"🛡", "Protected from abuse", "Orders are rate-limited per table so a prankster cannot flood your kitchen."},
			{"👀", "Staff stay in control", "QR orders appear on your POS for approval and billing, like any other order."},
			{"🖨", "Print-ready cards", "Print one card or all tables in a single go, sized to stand on the table."},
		},
		Steps:   []Point{{"1", "Print the cards", "Generate a QR card for each table and place it on the table."}, {"2", "Guest scans", "They see the menu, add items and place the order."}, {"3", "You serve and bill", "The kitchen cooks, the waiter serves, the cashier bills."}},
		Bullets: []string{"Shorter waiting for the first order", "Fewer wrong orders - guests type what they want", "Your waiters spend time serving, not writing", "Menu changes appear instantly - no reprinting menus"},
		FAQ: []QA{
			{"Do guests pay through the QR page?", "The QR page is for ordering. Payment is taken at the counter or by your staff the way you do today."},
			{"Can I change the menu or prices?", "Yes, anytime. The guest page always shows the current menu and marks items that are unavailable."},
			{"Will it work with a slow phone connection?", "The guest page is light and fast, built to open quickly on ordinary mobile data."},
		},
		Related: []string{"kitchen-display", "pos-billing", "customers-loyalty"}, BlogLinks: []string{"qr-code-ordering-for-restaurants"}},

	{Slug: "inventory", Kind: "product", Name: "Inventory & Purchasing", Icon: "📦",
		Short: "Stock, suppliers, purchase orders and wastage",
		Title: "Restaurant Inventory Management | MacBill System",
		Desc:  "Track ingredient stock, recipes, suppliers, purchase orders and wastage. Stock reduces automatically as you sell. Low-stock alerts included.",
		H1:    "Know what is in the store room before you run out",
		Lead:  "Stock goes down as you sell, purchases add it back, and low-stock alerts warn you in time. See where money is leaking through wastage and over-buying.",
		Points: []Point{
			{"📉", "Automatic deduction", "Link items to recipes and ingredients are reduced every time a dish is sold."},
			{"⚠", "Low-stock alerts", "Set a minimum for each item and get a warning before the shelf is empty."},
			{"🚚", "Suppliers & purchase orders", "Create a purchase order, receive the goods and the stock and payable are updated for you."},
			{"🗑", "Wastage tracking", "Record spoilage, breakage and staff meals so you can see the real cost of waste."},
			{"🏬", "Multi-location", "Keep separate stock per outlet and move stock between them with a transfer."},
			{"📋", "Stock take", "Count the shelf, enter the numbers, and the system shows the difference and adjusts."},
		},
		Steps:   []Point{{"1", "Add items & recipes", "List ingredients and what each dish uses."}, {"2", "Buy and receive", "Raise purchase orders and receive goods against them."}, {"3", "Watch the numbers", "See stock levels, usage and wastage in reports."}},
		Bullets: []string{"Every stock movement is recorded with who and when", "Spot theft and over-portioning early", "Supplier-wise purchase history", "Costing: see what each dish really costs you"},
		FAQ: []QA{
			{"Do I have to use recipes?", "No. You can track simple stock items only, or add recipes later for automatic deduction."},
			{"Can it track more than one outlet?", "Yes. Each branch has its own stock, with transfers between branches."},
			{"Does it update the accounts?", "Purchases and stock changes flow into your books automatically, so accounting stays in step."},
		},
		Related: []string{"accounting", "pos-billing", "reports-analytics"}, BlogLinks: []string{"reduce-food-wastage-with-inventory-tracking"}},

	{Slug: "accounting", Kind: "product", Name: "Accounting", Icon: "📒",
		Short: "Real double-entry books, built in",
		Title: "Restaurant Accounting Software | MacBill System",
		Desc:  "Built-in double-entry accounting for restaurants: sales, purchases, expenses, profit & loss, balance sheet and GST-ready reports without separate software.",
		H1:    "Accounting that updates itself every time you sell",
		Lead:  "Each sale, purchase and expense is posted to proper double-entry books automatically. Open the profit & loss or balance sheet whenever you want - no month-end scramble.",
		Points: []Point{
			{"⚙", "Automatic posting", "Sales, payments, purchases and payroll create the right entries for you."},
			{"📊", "Real statements", "Trial balance, profit & loss and balance sheet - the same reports your accountant expects."},
			{"🧾", "Tax-ready", "Tax collected on sales is tracked separately so filing is easier. Check final returns with your CA."},
			{"💸", "Expenses & payables", "Record expenses, pay suppliers and keep an eye on what you owe and what is owed to you."},
			{"🔒", "Period close", "Lock a finished month so nobody can change old numbers by accident."},
			{"📤", "Export anything", "Download reports to Excel or CSV and hand them to your accountant in one click."},
		},
		Steps:   []Point{{"1", "Run the business", "Bill, buy and pay people as normal."}, {"2", "Books update", "Entries are created automatically and balanced."}, {"3", "Review & share", "Open reports or send them to your accountant."}},
		Bullets: []string{"Chart of accounts ready for restaurants", "Manual journal entries when you need them", "Bank and cash tracking", "Every entry has an audit trail"},
		FAQ: []QA{
			{"Do I still need Tally or other software?", "Many owners can skip a separate tool for day-to-day books. Your accountant can still export the data if they prefer their own software."},
			{"Is it hard to use if I am not an accountant?", "No. Normal work posts entries for you. The accounting screens are there when you want to look."},
			{"Can I close a month?", "Yes. Closing a period blocks changes to that period."},
		},
		Related: []string{"inventory", "hr-payroll", "reports-analytics"}, BlogLinks: []string{"gst-billing-for-restaurants-simple-guide"}},

	{Slug: "hr-payroll", Kind: "product", Name: "HR & Payroll", Icon: "👥",
		Short: "Attendance, leave and salary slips",
		Title: "Restaurant Staff & Payroll Software | MacBill System",
		Desc:  "Manage restaurant staff: attendance check-in, leave requests, shifts and payroll with salary slips - connected to roles and accounting.",
		H1:    "Staff attendance, leave and salaries without the paperwork",
		Lead:  "Staff check in and out, request leave from their own login, and payroll turns attendance into salary slips - with the cost posted straight to your books.",
		Points: []Point{
			{"⏱", "Check-in & check-out", "Staff mark their own attendance. Late arrivals and worked hours are counted for you."},
			{"🌴", "Leave management", "Staff request leave, managers approve, and balances update automatically."},
			{"💼", "Payroll", "Generate payroll for the month with allowances, deductions and salary slips."},
			{"🔑", "Role-based access", "12 ready roles - owner, manager, cashier, waiter, chef and more - plus custom roles."},
			{"📍", "Optional location check", "Require staff to check in from the restaurant location if you want to."},
			{"📑", "Clear records", "A full history of attendance, leave and payments for every employee."},
		},
		Steps:   []Point{{"1", "Add staff", "Create employees and link each to a login."}, {"2", "Track the month", "Attendance and leave are recorded as the days go by."}, {"3", "Run payroll", "Review, approve and print salary slips."}},
		Bullets: []string{"Staff see only their own attendance and leave", "Managers approve leave in one click", "Salary cost flows into accounting", "Every important change is logged"},
		FAQ: []QA{
			{"Can staff use their own phone?", "Yes. The staff screen works in a phone browser."},
			{"Can I control what each person sees?", "Yes. Each role has its own permissions, and you can make custom roles."},
			{"Does payroll include deductions?", "Yes. You can set allowances and deductions and review before approving."},
		},
		Related: []string{"accounting", "pos-billing", "reports-analytics"}, BlogLinks: []string{"restaurant-staff-roles-and-permissions"}},

	{Slug: "delivery", Kind: "product", Name: "Delivery Management", Icon: "🛵",
		Short: "Assign drivers, track orders, cash on delivery",
		Title: "Restaurant Delivery Management | MacBill System",
		Desc:  "Manage your own delivery: take delivery orders, assign drivers, track status, handle cash on delivery and driver earnings.",
		H1:    "Run your own delivery without a mess of calls and chats",
		Lead:  "Take a delivery order at the counter, assign a driver, and follow it from the kitchen to the customer's door. Cash on delivery and driver earnings are tracked for you.",
		Points: []Point{
			{"📞", "Delivery orders in the POS", "Capture address, phone and delivery fee as part of the normal order."},
			{"🧑‍✈️", "Driver assignment", "Assign an available driver and see who has which order."},
			{"📍", "Status tracking", "Assigned, picked up, on the way and delivered - plus a public tracking link for the customer."},
			{"💵", "Cash on delivery", "Know how much cash each driver is carrying and settle it at the end of the shift."},
			{"💰", "Driver earnings", "Per-delivery pay is calculated automatically for each driver."},
			{"📈", "Delivery reports", "On-time rate, average time and revenue by area help you improve."},
		},
		Steps:   []Point{{"1", "Take the order", "Enter customer and address in the POS."}, {"2", "Assign a driver", "Pick a driver; the kitchen sends the food out."}, {"3", "Track to delivered", "Status updates and cash is settled."}},
		Bullets: []string{"Works with your own riders", "Customers can follow their order with a link", "Cash and earnings reconciled per driver", "Everything ties into sales and accounting"},
		FAQ: []QA{
			{"Does it connect to Zomato or Swiggy?", "MacBill manages your own delivery and orders you enter. Marketplace integrations are not part of this plan."},
			{"Can customers track their order?", "Yes. Each delivery gets a tracking link they can open on their phone."},
			{"How are drivers paid?", "You set the pay per delivery, and earnings are totalled for each driver."},
		},
		Related: []string{"pos-billing", "customers-loyalty", "reports-analytics"}},

	{Slug: "customers-loyalty", Kind: "product", Name: "Customers & Loyalty", Icon: "🎁",
		Short: "Customer history, points, gift cards, vouchers",
		Title: "Restaurant Loyalty & Gift Cards | MacBill System",
		Desc:  "Build repeat business: customer history, loyalty points, gift cards and vouchers, all linked to your POS bills.",
		H1:    "Turn first-time guests into regulars",
		Lead:  "Remember every customer, reward them with points, sell gift cards and run voucher offers - all from the same screen you bill on.",
		Points: []Point{
			{"📇", "Customer profiles", "Phone, birthday, order history and total spend in one place."},
			{"⭐", "Loyalty points", "Guests earn points on every bill and redeem them as payment."},
			{"🎁", "Gift cards", "Sell gift cards, track the balance and accept them as a payment mode."},
			{"🏷", "Vouchers", "Percentage or fixed-amount offers with limits and expiry dates."},
			{"🎂", "Birthday list", "See whose birthday is coming up, so you can invite them with an offer."},
			{"📊", "Who spends what", "Find your best customers and the ones who have not visited in a while."},
		},
		Steps:   []Point{{"1", "Capture the customer", "Add a name and phone at billing."}, {"2", "Reward them", "Points and vouchers apply automatically."}, {"3", "Bring them back", "Use the lists and reports to invite them again."}},
		Bullets: []string{"Search a customer by phone in seconds", "Points and gift cards work as payment methods", "Full history for every customer", "Reports on repeat visits and spend"},
		FAQ: []QA{
			{"Can I set how many points customers earn?", "Yes. The earn and redeem rules are in settings."},
			{"Do gift cards expire?", "You can set an expiry date, or leave them open."},
			{"Is customer data private?", "Only staff with the right role can see customer details."},
		},
		Related: []string{"pos-billing", "qr-ordering", "reports-analytics"}},

	{Slug: "reports-analytics", Kind: "product", Name: "Reports & Analytics", Icon: "📊",
		Short: "Sales, items, staff and profit at a glance",
		Title: "Restaurant Sales Reports & Analytics | MacBill System",
		Desc:  "See sales, best-selling items, payment modes, staff performance and profit in clear dashboards and exportable reports.",
		H1:    "See how your restaurant is really doing - today",
		Lead:  "A clear dashboard for the owner and detailed reports for the manager. Spot slow items, busy hours and leaks without digging through registers.",
		Points: []Point{
			{"📅", "Daily to yearly", "Sales by day, week, month or any custom dates."},
			{"🏆", "Best & worst items", "Learn what sells, what does not and what makes the most profit."},
			{"💳", "Payment modes", "Cash, card and UPI totals ready for end-of-day closing."},
			{"👨‍🍳", "Staff performance", "Orders and sales per waiter and cashier."},
			{"🏬", "Branch comparison", "Compare outlets side by side from one login."},
			{"⬇", "Export", "Download any report as Excel or CSV."},
		},
		Steps:   []Point{{"1", "Work as usual", "Every bill feeds the reports."}, {"2", "Open the dashboard", "See today at a glance."}, {"3", "Dig deeper", "Filter, compare and export."}},
		Bullets: []string{"Live numbers, not yesterday's", "End-of-day closing summary", "Taxes and discounts reports", "Permission-controlled: only the right people see profit"},
		FAQ: []QA{
			{"Can I see all my outlets together?", "Yes. Head-office users can view every branch or switch to one."},
			{"Can I get reports on my phone?", "Yes. The dashboard works in a phone browser."},
			{"Can my accountant get the data?", "Export to Excel or CSV in a click."},
		},
		Related: []string{"accounting", "inventory", "pos-billing"}},
}

var solutions = []*Detail{
	{Slug: "restaurants", Kind: "solution", Name: "Restaurants", Icon: "🍽", Short: "Dine-in with tables, kitchen and waiters",
		Title: "Billing Software for Restaurants | MacBill System", Desc: "Table management, waiter ordering, kitchen display and fast billing - a complete system for dine-in restaurants.",
		H1: "Billing software built for busy dine-in restaurants", Lead: "Floor plan, waiters, kitchen and cashier all work from the same live orders - so service stays smooth even when every table is full.",
		Points:  []Point{{"🪑", "Table management", "See free, running and billed tables at a glance and move orders between them."}, {"🧑‍🍳", "Waiter to kitchen", "Waiters send orders directly; the kitchen sees them instantly."}, {"🧾", "Split & merge bills", "Split by item or amount, merge tables and take mixed payments."}, {"📱", "QR ordering", "Let guests order from the table when you are short-staffed."}},
		Bullets: []string{"Reservations and table status", "Service charge and tips", "Role-based staff access", "End-of-day cash closing"},
		FAQ:     []QA{{"Can I manage many tables?", "Yes. Create sections and tables, and see their status live."}, {"Do waiters need training?", "A new waiter can usually learn the order screen in minutes."}},
		Related: []string{"pos-billing", "kitchen-display", "qr-ordering"}},
	{Slug: "cafes", Kind: "solution", Name: "Cafés", Icon: "☕", Short: "Quick counter billing and loyalty",
		Title: "Café Billing Software & POS | MacBill System", Desc: "Quick counter billing, barcode scanning, loyalty points and simple inventory for cafés and coffee shops.",
		H1: "A fast counter POS for cafés and coffee shops", Lead: "Short queues, quick bills and a loyalty programme that brings people back for their next cup.",
		Points:  []Point{{"⚡", "Counter speed", "One-tap items, modifiers like size and milk, and quick payment."}, {"⭐", "Loyalty", "Points and vouchers keep regulars coming."}, {"📦", "Simple stock", "Track beans, milk and packaged items."}, {"🖨", "Small slips", "Neat 58mm or 80mm receipts with your branding."}},
		Bullets: []string{"Takeaway-first workflow", "Barcode scanning for packaged goods", "Customer display option", "Daily sales summary"},
		FAQ:     []QA{{"Can I use it on a tablet?", "Yes, it works in any modern browser."}, {"Can I add item options?", "Yes - sizes, extras and notes are supported."}},
		Related: []string{"pos-billing", "customers-loyalty", "inventory"}},
	{Slug: "cloud-kitchens", Kind: "solution", Name: "Cloud Kitchens", Icon: "🥡", Short: "Delivery-first kitchens and multiple brands",
		Title: "Cloud Kitchen Software | MacBill System", Desc: "Manage delivery orders, a kitchen display, stock and drivers for a cloud kitchen with no dining area.",
		H1: "Run a delivery-only kitchen with less chaos", Lead: "Capture delivery orders, show them on the kitchen screen, hand them to a rider and keep an eye on stock and margins.",
		Points:  []Point{{"🛵", "Delivery first", "Delivery orders with address, fee and driver assignment."}, {"📺", "Kitchen screen", "Every order on one screen in the order it must be cooked."}, {"📦", "Stock control", "Ingredient tracking and wastage reports protect margin."}, {"📊", "Margins", "See sales, cost and profit by day and by item."}},
		Bullets: []string{"No dining module needed", "Rider cash tracking", "Multiple outlets from one login", "Fast onboarding"},
		FAQ:     []QA{{"Does it handle several brands?", "You can create separate menus and outlets and compare them."}, {"Can customers track orders?", "Yes, with a tracking link."}},
		Related: []string{"delivery", "kitchen-display", "inventory"}},
	{Slug: "restaurant-chains", Kind: "solution", Name: "Restaurant Chains", Icon: "🏢", Short: "Many outlets, one dashboard",
		Title: "Multi-Outlet Restaurant Software | MacBill System", Desc: "Manage several restaurants from one login: shared menu, per-branch stock, central reports and role-based access.",
		H1: "One dashboard for all your outlets", Lead: "Share the menu, compare branches and keep each outlet's staff and stock separate - with head-office control.",
		Points:  []Point{{"🏬", "Branch control", "Outlet staff see only their own branch; head office sees everything."}, {"🍴", "Shared menu", "Update an item once and every outlet gets it."}, {"📊", "Compare outlets", "Sales, items and staff side by side."}, {"🔐", "Permissions", "Fine-grained roles protect shared settings and data."}},
		Bullets: []string{"Per-branch stock and cash", "Central accounting and reports", "Branch-wise taxes and settings", "Audit trail of important actions"},
		FAQ:     []QA{{"How many outlets can I add?", "The Chain plan covers up to five branches. For more, contact us."}, {"Can branch managers change the menu?", "Only if you give them that permission."}},
		Related: []string{"reports-analytics", "accounting", "hr-payroll"}},
	{Slug: "bakeries-sweet-shops", Kind: "solution", Name: "Bakeries & Sweet Shops", Icon: "🧁", Short: "Weight, packaged items and counters",
		Title: "Bakery & Sweet Shop Billing | MacBill System", Desc: "Counter billing for bakeries and sweet shops with barcode scanning, packaged and loose items, stock and gift cards.",
		H1: "Billing for bakeries and sweet shops", Lead: "Scan packaged items, bill loose items quickly and keep track of stock and festive-season rush.",
		Points:  []Point{{"🔍", "Barcode scanning", "Scan packaged goods; search for the rest."}, {"🎁", "Gift cards & vouchers", "Perfect for festivals and gifting."}, {"📦", "Stock", "Know what is selling and what is expiring."}, {"🧾", "Quick bills", "Short queues even at peak hours."}},
		Bullets: []string{"Works with USB barcode scanners", "Customer loyalty", "Daily sales report", "Multiple counters"},
		FAQ:     []QA{{"Can I use a barcode scanner?", "Yes. A normal USB scanner works like a keyboard and fills the search box."}, {"Can I run offers?", "Yes, with vouchers and discounts."}},
		Related: []string{"pos-billing", "inventory", "customers-loyalty"}},
}

type Post struct {
	Slug, Title, Desc, Date, Read string
	Intro                         string
	Sections                      []Section
	Related                       []string // product slugs
	FAQ                           []QA
	Root                          bool // guide pages live at /slug instead of /blog/slug
}

var guides []*Post

func (p *Post) Path() string {
	if p.Root {
		return "/" + p.Slug
	}
	return "/blog/" + p.Slug
}

type Section struct {
	H    string
	Body []string
	List []string
}

var posts = []*Post{
	{Slug: "how-to-choose-restaurant-billing-software", Date: "2026-09-12", Read: "6 min",
		Title: "How to Choose Restaurant Billing Software: A Checklist",
		Desc:  "A simple checklist for choosing restaurant billing software: ease of use, printing, kitchen flow, reports, support and total cost.",
		Intro: "Picking billing software is a decision you will live with every day. A good system makes service faster; a bad one slows every shift. Use this checklist to compare options calmly.",
		Sections: []Section{
			{H: "1. Can a new person learn it quickly?", Body: []string{"Staff turnover is normal in restaurants. If a new waiter or cashier needs days of training, you lose money every time someone joins. Ask for a live demo and let your own staff try taking an order."}},
			{H: "2. Does it fit your way of working?", Body: []string{"A dine-in restaurant needs tables, waiters and a kitchen flow. A café needs speed at the counter. A cloud kitchen needs delivery tracking. Make sure the software handles your main workflow without workarounds."}},
			{H: "3. Printing and hardware", Body: []string{"Check that it prints properly on the receipt printer you already own - 58mm or 80mm - and that the bill shows everything you need, such as tax details, payment mode and the cashier name."}},
			{H: "4. Reports you will actually use", Body: []string{"You should be able to see today's sales, best-selling items and payment totals without help. Ask whether you can export data for your accountant."}},
			{H: "5. Control over staff access", Body: []string{"Not everyone should see profit or cancel bills. Look for role-based permissions and an audit trail that shows who did what."}},
			{H: "6. Support and the real cost", Body: []string{"Ask about training, support hours and how renewals work. Compare the total cost for a year rather than the headline monthly price, and check what is included."}},
		},
		Related: []string{"pos-billing", "reports-analytics"}},
	{Slug: "gst-billing-for-restaurants-simple-guide", Date: "2026-09-18", Read: "5 min",
		Title: "GST Billing for Restaurants: A Simple Guide for Owners",
		Desc:  "What a restaurant bill should show, how tax is presented and how software helps keep GST records tidy. Not tax advice.",
		Intro: "Tax rules change and depend on your situation, so always confirm the current rules and rates with your chartered accountant. This guide explains the practical side: what a clean bill and tidy records look like.",
		Sections: []Section{
			{H: "What a bill should clearly show", Body: []string{"A restaurant bill is easier to defend and easier to file when it is complete and consistent."}, List: []string{"Your business name, address and tax registration number", "A unique, sequential bill number and the date and time", "Items with quantity and price", "Discounts and service charge, if any", "Tax shown separately from the item total", "The final amount and the payment mode"}},
			{H: "Why consistent numbering matters", Body: []string{"Sequential bill numbers make it easy to prove that no bill is missing. Good software generates them for you and records every void and refund with a reason."}},
			{H: "Keep tax separate from sales", Body: []string{"When tax collected is tracked in its own account, preparing returns is simpler. MacBill System posts tax on every sale to a separate ledger automatically."}},
			{H: "Hand clean data to your accountant", Body: []string{"At the end of the month, export your sales and tax reports to Excel or CSV and send them across. Fewer questions, faster filing."}},
		},
		Related: []string{"pos-billing", "accounting"}},
	{Slug: "reduce-food-wastage-with-inventory-tracking", Date: "2026-09-25", Read: "5 min",
		Title: "How to Reduce Food Wastage with Simple Inventory Tracking",
		Desc:  "Practical ways restaurants cut wastage: track stock, set minimums, record spoilage and review usage against sales.",
		Intro: "Wastage is a quiet cost. It rarely shows up as one big loss; it leaks away a little at a time. Tracking is the first step to stopping it.",
		Sections: []Section{
			{H: "Start by counting", Body: []string{"You cannot manage what you do not measure. Begin with a weekly count of your most expensive and most perishable items - proteins, dairy and fresh produce."}},
			{H: "Set a minimum level", Body: []string{"A minimum stock level for each item gives you a warning before you run out, and stops panic buying at higher prices."}},
			{H: "Record spoilage and staff meals", Body: []string{"When wasted items are recorded with a reason, patterns appear: too much prepared in advance, wrong storage, or over-ordering."}},
			{H: "Compare usage to sales", Body: []string{"If you sold 40 portions but used enough ingredients for 55, something is off. Linking recipes to stock lets the system show you the gap."}},
			{H: "Review weekly", Body: []string{"Spend ten minutes each week looking at the wastage and low-stock reports. Small fixes - smaller batches, better labelling, tighter ordering - add up quickly."}},
		},
		Related: []string{"inventory", "reports-analytics"}},
	{Slug: "qr-code-ordering-for-restaurants", Date: "2026-10-02", Read: "4 min",
		Title: "QR Code Ordering for Restaurants: Is It Right for You?",
		Desc:  "How QR table ordering works, when it helps, and how to introduce it without losing the personal touch.",
		Intro: "QR ordering lets guests scan a code on the table and order from their phone. Done well, it speeds up service; done carelessly, it can feel cold. Here is how to get the balance right.",
		Sections: []Section{
			{H: "When it helps most", Body: []string{"Busy hours, short staffing and large tables where one waiter cannot take every order quickly. Guests who know what they want can order immediately."}},
			{H: "Keep the human touch", Body: []string{"Greet guests, explain the card in a sentence, and offer to take the order yourself if they prefer. QR is an option, not a replacement for hospitality."}},
			{H: "Make the menu shine", Body: []string{"Use clear item names, honest descriptions and mark unavailable items. The menu on the guest page updates instantly when you change it in the system."}},
			{H: "Protect your kitchen", Body: []string{"Rate limits per table prevent accidental or prank floods, and staff see QR orders on the POS before billing."}},
		},
		Related: []string{"qr-ordering", "kitchen-display"}},
	{Slug: "kitchen-display-system-vs-paper-kot", Date: "2026-10-06", Read: "4 min",
		Title: "Kitchen Display System vs Paper KOT: Which Is Better?",
		Desc:  "Compare paper kitchen order tickets with a kitchen display screen, and when to use both.",
		Intro: "Many kitchens still run on printed tickets. A kitchen display is a modern alternative - but it is not all or nothing.",
		Sections: []Section{
			{H: "Where paper tickets win", Body: []string{"They are simple, familiar and need no screen. Some chefs like a physical slip to hold or stick on the pass."}},
			{H: "Where a screen wins", Body: []string{"A screen shows orders in sequence, changes colour as they wait, and lets the chef mark items ready. That ready signal can alert the waiter automatically, so food does not sit and go cold."}},
			{H: "Use both while you transition", Body: []string{"MacBill can auto-print kitchen tickets and show the screen at the same time. Start with both, then drop the printer when your team is comfortable."}},
		},
		Related: []string{"kitchen-display", "pos-billing"}},
	{Slug: "restaurant-staff-roles-and-permissions", Date: "2026-10-09", Read: "5 min",
		Title: "Restaurant Staff Roles and Permissions: Who Does What?",
		Desc:  "A practical guide to giving restaurant staff the right access in your billing system, and preventing mistakes and misuse.",
		Intro: "Most billing problems are not caused by bad software but by too much access. Giving each person only the access they need protects your money and your staff.",
		Sections: []Section{
			{H: "Waiters", Body: []string{"Take orders and send them to the kitchen. They usually should not print bills or see reports."}},
			{H: "Cashiers and POS operators", Body: []string{"Take payment and print bills. A POS operator role can be limited to seeing and printing only their own bills."}},
			{H: "Managers", Body: []string{"Approve voids, discounts and refunds; see reports; manage stock and staff."}},
			{H: "Owners and head office", Body: []string{"See everything across branches, including profit and accounting. Keep this group small."}},
			{H: "Keep an audit trail", Body: []string{"Whatever the permissions, an audit log of who changed what makes problems easy to trace and discourages misuse."}},
		},
		Related: []string{"hr-payroll", "pos-billing"}},
}

func findProduct(slug string) *Detail {
	for _, d := range products {
		if d.Slug == slug {
			return d
		}
	}
	for _, d := range solutions {
		if d.Slug == slug {
			return d
		}
	}
	return nil
}
func findPost(slug string) *Post {
	for _, p := range posts {
		if p.Slug == slug {
			return p
		}
	}
	return nil
}
