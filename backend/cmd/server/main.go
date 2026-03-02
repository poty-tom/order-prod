package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

type server struct {
	db     *sql.DB
	amqpCh *amqp.Channel
}

type product struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	PriceCents        int64      `json:"priceCents"`
	Stock             int64      `json:"stock"`
	SalePercent       int64      `json:"salePercent"`
	SaleStart         *time.Time `json:"saleStart,omitempty"`
	SaleEnd           *time.Time `json:"saleEnd,omitempty"`
	CurrentPriceCents int64      `json:"currentPriceCents"`
	OnSale            bool       `json:"onSale"`
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PriceCents  int64   `json:"priceCents"`
	Stock       int64   `json:"stock"`
	SalePercent int64   `json:"salePercent"`
	SaleStart   *string `json:"saleStart"`
	SaleEnd     *string `json:"saleEnd"`
}

type addCartItemRequest struct {
	UserID    string `json:"userId"`
	ProductID int64  `json:"productId"`
	Quantity  int64  `json:"quantity"`
}

type removeCartItemRequest struct {
	UserID    string `json:"userId"`
	ProductID int64  `json:"productId"`
}

type checkoutRequest struct {
	UserID string `json:"userId"`
}

type cartItemResponse struct {
	ProductID          int64  `json:"productId"`
	Name               string `json:"name"`
	Quantity           int64  `json:"quantity"`
	PriceCents         int64  `json:"priceCents"`
	DiscountPercent    int64  `json:"discountPercent"`
	CurrentPriceCents  int64  `json:"currentPriceCents"`
	SubtotalPriceCents int64  `json:"subtotalPriceCents"`
}

type cartResponse struct {
	UserID     string             `json:"userId"`
	Items      []cartItemResponse `json:"items"`
	TotalCents int64              `json:"totalCents"`
}

type checkoutResponse struct {
	OrderID    int64  `json:"orderId"`
	UserID     string `json:"userId"`
	TotalCents int64  `json:"totalCents"`
	Status     string `json:"status"`
}

type orderEvent struct {
	OrderID    int64  `json:"orderId"`
	UserID     string `json:"userId"`
	TotalCents int64  `json:"totalCents"`
	CreatedAt  string `json:"createdAt"`
}

func main() {
	mysqlDSN := envOrDefault("MYSQL_DSN", "root:root@tcp(mysql:3306)/shop?parseTime=true")
	rabbitURL := envOrDefault("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	port := envOrDefault("PORT", "8080")

	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := waitForDB(ctx, db); err != nil {
		log.Fatalf("mysql not ready: %v", err)
	}
	if err := ensureSchema(db); err != nil {
		log.Fatalf("schema init: %v", err)
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("create rabbit channel: %v", err)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare("orders.confirmed", true, false, false, false, nil); err != nil {
		log.Fatalf("declare queue: %v", err)
	}

	s := &server{db: db, amqpCh: ch}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/products", s.products)
	mux.HandleFunc("/admin/products", s.adminProducts)
	mux.HandleFunc("/cart", s.getCart)
	mux.HandleFunc("/cart/items", s.cartItems)
	mux.HandleFunc("/checkout", s.checkout)

	handler := corsMiddleware(mux)
	log.Printf("api listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func waitForDB(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func ensureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS products (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price_cents BIGINT NOT NULL,
			stock BIGINT NOT NULL,
			sale_percent BIGINT NOT NULL DEFAULT 0,
			sale_start DATETIME NULL,
			sale_end DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS carts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(100) NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS cart_items (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			cart_id BIGINT NOT NULL,
			product_id BIGINT NOT NULL,
			quantity BIGINT NOT NULL,
			UNIQUE KEY uniq_cart_product (cart_id, product_id),
			FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
			FOREIGN KEY (product_id) REFERENCES products(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS orders (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(100) NOT NULL,
			status VARCHAR(30) NOT NULL,
			total_cents BIGINT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			order_id BIGINT NOT NULL,
			product_id BIGINT NOT NULL,
			quantity BIGINT NOT NULL,
			unit_price_cents BIGINT NOT NULL,
			discount_percent BIGINT NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
			FOREIGN KEY (product_id) REFERENCES products(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) adminProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" || req.PriceCents <= 0 || req.Stock < 0 {
		http.Error(w, "invalid product fields", http.StatusBadRequest)
		return
	}
	if req.SalePercent < 0 || req.SalePercent > 90 {
		http.Error(w, "salePercent must be between 0 and 90", http.StatusBadRequest)
		return
	}

	var saleStart, saleEnd *time.Time
	if req.SalePercent > 0 {
		if req.SaleStart == nil || req.SaleEnd == nil {
			http.Error(w, "saleStart and saleEnd are required when salePercent > 0", http.StatusBadRequest)
			return
		}
		start, err := time.Parse(time.RFC3339, *req.SaleStart)
		if err != nil {
			http.Error(w, "saleStart must be RFC3339", http.StatusBadRequest)
			return
		}
		end, err := time.Parse(time.RFC3339, *req.SaleEnd)
		if err != nil {
			http.Error(w, "saleEnd must be RFC3339", http.StatusBadRequest)
			return
		}
		if !end.After(start) {
			http.Error(w, "saleEnd must be after saleStart", http.StatusBadRequest)
			return
		}
		saleStart = &start
		saleEnd = &end
	}

	res, err := s.db.Exec(`INSERT INTO products(name, description, price_cents, stock, sale_percent, sale_start, sale_end)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Description, req.PriceCents, req.Stock, req.SalePercent, saleStart, saleEnd)
	if err != nil {
		http.Error(w, fmt.Sprintf("insert product: %v", err), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *server) products(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.db.Query(`SELECT id, name, description, price_cents, stock, sale_percent, sale_start, sale_end,
		CASE
			WHEN sale_percent > 0 AND sale_start IS NOT NULL AND sale_end IS NOT NULL AND NOW() BETWEEN sale_start AND sale_end
			THEN ROUND(price_cents * (100 - sale_percent) / 100)
			ELSE price_cents
		END AS current_price,
		CASE
			WHEN sale_percent > 0 AND sale_start IS NOT NULL AND sale_end IS NOT NULL AND NOW() BETWEEN sale_start AND sale_end
			THEN 1 ELSE 0
		END AS on_sale
		FROM products ORDER BY id DESC`)
	if err != nil {
		http.Error(w, fmt.Sprintf("query products: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := make([]product, 0)
	for rows.Next() {
		var p product
		var start, end sql.NullTime
		var onSale int64
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.PriceCents,
			&p.Stock,
			&p.SalePercent,
			&start,
			&end,
			&p.CurrentPriceCents,
			&onSale,
		); err != nil {
			http.Error(w, fmt.Sprintf("scan products: %v", err), http.StatusInternalServerError)
			return
		}
		if start.Valid {
			p.SaleStart = &start.Time
		}
		if end.Valid {
			p.SaleEnd = &end.Time
		}
		p.OnSale = onSale == 1
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	resp, err := s.loadCart(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) cartItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.addCartItem(w, r)
	case http.MethodDelete:
		s.removeCartItem(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) addCartItem(w http.ResponseWriter, r *http.Request) {
	var req addCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserID) == "" || req.ProductID <= 0 || req.Quantity <= 0 {
		http.Error(w, "invalid fields", http.StatusBadRequest)
		return
	}

	cartID, err := s.findOrCreateCart(req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find/create cart: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = s.db.Exec(`INSERT INTO cart_items(cart_id, product_id, quantity)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)`, cartID, req.ProductID, req.Quantity)
	if err != nil {
		http.Error(w, fmt.Sprintf("upsert cart item: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := s.loadCart(req.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) removeCartItem(w http.ResponseWriter, r *http.Request) {
	var req removeCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserID) == "" || req.ProductID <= 0 {
		http.Error(w, "invalid fields", http.StatusBadRequest)
		return
	}

	_, err := s.db.Exec(`DELETE ci FROM cart_items ci
		JOIN carts c ON c.id = ci.cart_id
		WHERE c.user_id = ? AND ci.product_id = ?`, req.UserID, req.ProductID)
	if err != nil {
		http.Error(w, fmt.Sprintf("delete cart item: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := s.loadCart(req.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) checkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	resp, err := s.checkoutTx(req.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "cart is empty", http.StatusBadRequest)
			return
		}
		if strings.HasPrefix(err.Error(), "stock shortage") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("checkout failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *server) checkoutTx(userID string) (*checkoutResponse, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var cartID int64
	if err := tx.QueryRow(`SELECT id FROM carts WHERE user_id = ? FOR UPDATE`, userID).Scan(&cartID); err != nil {
		return nil, err
	}

	type line struct {
		ProductID       int64
		Name            string
		Quantity        int64
		PriceCents      int64
		Stock           int64
		DiscountPercent int64
		CurrentPrice    int64
	}

	rows, err := tx.Query(`SELECT p.id, p.name, ci.quantity, p.price_cents, p.stock,
		CASE
			WHEN p.sale_percent > 0 AND p.sale_start IS NOT NULL AND p.sale_end IS NOT NULL AND NOW() BETWEEN p.sale_start AND p.sale_end
			THEN p.sale_percent ELSE 0
		END AS discount_percent,
		CASE
			WHEN p.sale_percent > 0 AND p.sale_start IS NOT NULL AND p.sale_end IS NOT NULL AND NOW() BETWEEN p.sale_start AND p.sale_end
			THEN ROUND(p.price_cents * (100 - p.sale_percent) / 100)
			ELSE p.price_cents
		END AS current_price
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = ? FOR UPDATE`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]line, 0)
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.ProductID, &l.Name, &l.Quantity, &l.PriceCents, &l.Stock, &l.DiscountPercent, &l.CurrentPrice); err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}

	total := int64(0)
	for _, item := range items {
		if item.Stock < item.Quantity {
			return nil, fmt.Errorf("stock shortage: product_id=%d remaining=%d requested=%d", item.ProductID, item.Stock, item.Quantity)
		}
		total += item.CurrentPrice * item.Quantity
	}

	res, err := tx.Exec(`INSERT INTO orders(user_id, status, total_cents) VALUES (?, 'CONFIRMED', ?)`, userID, total)
	if err != nil {
		return nil, err
	}
	orderID, _ := res.LastInsertId()

	for _, item := range items {
		if _, err := tx.Exec(`INSERT INTO order_items(order_id, product_id, quantity, unit_price_cents, discount_percent)
			VALUES (?, ?, ?, ?, ?)`, orderID, item.ProductID, item.Quantity, item.PriceCents, item.DiscountPercent); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE products SET stock = stock - ? WHERE id = ?`, item.Quantity, item.ProductID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(`DELETE FROM cart_items WHERE cart_id = ?`, cartID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	event := orderEvent{
		OrderID:    orderID,
		UserID:     userID,
		TotalCents: total,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.publishOrderEvent(event); err != nil {
		log.Printf("warn: order created but event publish failed: %v", err)
	}

	return &checkoutResponse{OrderID: orderID, UserID: userID, TotalCents: total, Status: "CONFIRMED"}, nil
}

func (s *server) publishOrderEvent(event orderEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.amqpCh.PublishWithContext(context.Background(), "", "orders.confirmed", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	})
}

func (s *server) findOrCreateCart(userID string) (int64, error) {
	_, err := s.db.Exec(`INSERT INTO carts(user_id) VALUES (?) ON DUPLICATE KEY UPDATE user_id = VALUES(user_id)`, userID)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM carts WHERE user_id = ?`, userID).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *server) loadCart(userID string) (*cartResponse, error) {
	cartID, err := s.findOrCreateCart(userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT p.id, p.name, ci.quantity, p.price_cents,
		CASE
			WHEN p.sale_percent > 0 AND p.sale_start IS NOT NULL AND p.sale_end IS NOT NULL AND NOW() BETWEEN p.sale_start AND p.sale_end
			THEN p.sale_percent ELSE 0
		END AS discount_percent,
		CASE
			WHEN p.sale_percent > 0 AND p.sale_start IS NOT NULL AND p.sale_end IS NOT NULL AND NOW() BETWEEN p.sale_start AND p.sale_end
			THEN ROUND(p.price_cents * (100 - p.sale_percent) / 100)
			ELSE p.price_cents
		END AS current_price
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = ?
		ORDER BY ci.id DESC`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]cartItemResponse, 0)
	total := int64(0)
	for rows.Next() {
		var it cartItemResponse
		if err := rows.Scan(&it.ProductID, &it.Name, &it.Quantity, &it.PriceCents, &it.DiscountPercent, &it.CurrentPriceCents); err != nil {
			return nil, err
		}
		it.SubtotalPriceCents = int64(math.Round(float64(it.CurrentPriceCents * it.Quantity)))
		total += it.SubtotalPriceCents
		items = append(items, it)
	}

	return &cartResponse{UserID: userID, Items: items, TotalCents: total}, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, strconv.FormatInt(int64(status), 10), status)
	}
}
