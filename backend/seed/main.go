// Jednorazowy seeder danych demo dla katalogu: kategoria, sklep, produkty, zdjęcia.
// Uruchom: cd backend/seed && go run main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://admin:npg_TokODPY2bS4f@ep-still-band-aiqm10i0-pooler.c-4.us-east-1.aws.neon.tech/flipthetable?sslmode=require"
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	sellerID := seedSeller(ctx, conn)
	shopID := seedShop(ctx, conn, sellerID)
	categoryIDs := seedCategories(ctx, conn)
	seedListings(ctx, conn, shopID, categoryIDs)

	fmt.Println("Seed zakończony pomyślnie.")
}

func seedSeller(ctx context.Context, conn *pgx.Conn) string {
	var id string
	err := conn.QueryRow(ctx, `SELECT id FROM users WHERE email = 'demo-seller@flipthetable.local'`).Scan(&id)
	if err == nil {
		return id
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("demoSellerPass123"), bcrypt.DefaultCost)
	err = conn.QueryRow(ctx, `
		INSERT INTO users (email, username, full_name, password_hash, is_seller)
		VALUES ('demo-seller@flipthetable.local', 'demo_seller', 'Demo Seller', $1, TRUE)
		RETURNING id
	`, string(hash)).Scan(&id)
	if err != nil {
		fmt.Println("seedSeller error:", err)
		os.Exit(1)
	}
	return id
}

func seedShop(ctx context.Context, conn *pgx.Conn, ownerID string) string {
	var id string
	err := conn.QueryRow(ctx, `SELECT id FROM shops WHERE slug = 'demo-shop'`).Scan(&id)
	if err == nil {
		return id
	}

	err = conn.QueryRow(ctx, `
		INSERT INTO shops (owner_id, name, slug, description)
		VALUES ($1, 'Demo Handmade Shop', 'demo-shop', 'Ręcznie robione przedmioty na potrzeby demo katalogu')
		RETURNING id
	`, ownerID).Scan(&id)
	if err != nil {
		fmt.Println("seedShop error:", err)
		os.Exit(1)
	}
	return id
}

func seedCategories(ctx context.Context, conn *pgx.Conn) map[string]string {
	categories := []struct {
		Name string
		Slug string
	}{
		{"Biżuteria", "bizuteria"},
		{"Dekoracje domu", "dekoracje-domu"},
		{"Sztuka", "sztuka"},
		{"Vintage", "vintage"},
		{"Akcesoria", "akcesoria"},
	}

	ids := map[string]string{}
	for i, c := range categories {
		var id string
		err := conn.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, c.Slug).Scan(&id)
		if err == nil {
			ids[c.Slug] = id
			continue
		}
		err = conn.QueryRow(ctx, `
			INSERT INTO categories (name, slug, sort_order)
			VALUES ($1, $2, $3)
			RETURNING id
		`, c.Name, c.Slug, i).Scan(&id)
		if err != nil {
			fmt.Println("seedCategories error:", err)
			os.Exit(1)
		}
		ids[c.Slug] = id
	}
	return ids
}

func seedListings(ctx context.Context, conn *pgx.Conn, shopID string, categoryIDs map[string]string) {
	var existing int
	conn.QueryRow(ctx, `SELECT COUNT(*) FROM listings WHERE shop_id = $1`, shopID).Scan(&existing)
	if existing > 0 {
		fmt.Println("Produkty już istnieją, pomijam.")
		return
	}

	listings := []struct {
		Title       string
		Description string
		Price       string
		Quantity    int
		CategorySlug string
		PhotoSeed   string
	}{
		{"Ręcznie robiony naszyjnik z bursztynu", "Unikalny naszyjnik z polskiego bursztynu bałtyckiego, oprawiony w srebro.", "189.99", 5, "bizuteria", "necklace"},
		{"Kolczyki srebrne minimalistyczne", "Proste, eleganckie kolczyki idealne na co dzień.", "79.00", 12, "bizuteria", "earrings"},
		{"Świecznik ceramiczny ręcznie malowany", "Ceramiczny świecznik w stylu boho, malowany ręcznie.", "65.50", 8, "dekoracje-domu", "candleholder"},
		{"Makatka makrama na ścianę", "Duża makatka makrama, bawełniany sznurek, drewniany drążek.", "145.00", 4, "dekoracje-domu", "macrame"},
		{"Obraz akrylowy abstrakcja", "Oryginalny obraz akrylowy na płótnie 50x70cm.", "320.00", 1, "sztuka", "painting"},
		{"Grafika A3 w stylu retro", "Limitowana grafika drukowana na papierze archiwalnym.", "89.00", 20, "sztuka", "print"},
		{"Vintage torebka skórzana lata 70", "Oryginalna skórzana torebka w dobrym stanie, lata 70.", "210.00", 1, "vintage", "bag"},
		{"Vintage zegarek mechaniczny", "Zabytkowy zegarek mechaniczny po renowacji.", "450.00", 2, "vintage", "watch"},
		{"Torba na zakupy z bawełny organicznej", "Ekologiczna, wytrzymała torba wielokrotnego użytku.", "39.99", 30, "akcesoria", "totebag"},
		{"Portfel skórzany ręcznej roboty", "Portfel z naturalnej skóry, ręcznie szyty.", "129.00", 10, "akcesoria", "wallet"},
	}

	for _, l := range listings {
		var listingID string
		catID := categoryIDs[l.CategorySlug]
		err := conn.QueryRow(ctx, `
			INSERT INTO listings (shop_id, category_id, title, description, price, quantity)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, shopID, catID, l.Title, l.Description, l.Price, l.Quantity).Scan(&listingID)
		if err != nil {
			fmt.Println("seedListings insert error:", err)
			continue
		}

		photoURL := fmt.Sprintf("https://picsum.photos/seed/%s/600/600", l.PhotoSeed)
		_, err = conn.Exec(ctx, `
			INSERT INTO listing_photos (listing_id, url, alt_text, is_primary, sort_order)
			VALUES ($1, $2, $3, TRUE, 0)
		`, listingID, photoURL, l.Title)
		if err != nil {
			fmt.Println("seedListings photo error:", err)
		}
	}

	fmt.Printf("Dodano %d produktów.\n", len(listings))
}
