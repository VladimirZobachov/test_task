package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Product struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Price float64
}

type DishResponse struct {
	Products []Product `json:"products"`
	Price    float64   `json:"price"`
}

func main() {
	// Connect to the database
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/test_task")
	if err != nil {
		log.Fatalf("Error connection to mysql server: %s\n", err)
	}
	defer db.Close()

	http.HandleFunc("/constructor/", func(w http.ResponseWriter, r *http.Request) {
		dishConstructorHandler(w, r, db)
	})
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func dishConstructorHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ingredientCodes := strings.TrimPrefix(r.URL.Path, "/constructor/")
	if ingredientCodes == "" {
		http.Error(w, "Ingredient codes are required", http.StatusBadRequest)
		return
	}

	dishes, err := generateDishCombinations(db, ingredientCodes)
	if err != nil {
		http.Error(w, "Failed to generate combinations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dishes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func generateDishCombinations(db *sql.DB, codes string) ([]DishResponse, error) {

	var dishes []DishResponse
	var products []Product

	for _, code := range codes {
		var product Product
		var productsOfType []Product

		query := `
		SELECT it.title, i.title, i.price
		FROM ingredient i
		INNER JOIN ingredient_type it ON i.type_id = it.id
		WHERE it.code = ?
		`
		rows, err := db.Query(query, string(code))
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			if err := rows.Scan(&product.Type, &product.Value, &product.Price); err != nil {
				return nil, err
			}
			productsOfType = append(productsOfType, product)
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
		rows.Close()

		for _, prod := range productsOfType {
			products = append(products, prod)
		}
	}

	if len(products) > 0 {
		var total float64
		for _, p := range products {
			total += p.Price
		}
		dishes = append(dishes, DishResponse{Products: products, Price: total})
	}

	return dishes, nil
}
