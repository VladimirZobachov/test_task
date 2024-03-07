package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // Import your database driver as needed
	"log"
	"net/http"
	"strings"
)

// Ingredient represents a single ingredient in a dish.
type Ingredient struct {
	Type  string  `json:"type"`
	Value string  `json:"value"`
	Price float64 `json:"price"`
}

// Product represents a combination of ingredients.
type Product struct {
	Ingredients []Ingredient `json:"ingredients"`
	Price       float64      `json:"price"`
}

// Helper function to get ingredients for a specific code from the database.
func getIngredientsByCode(db *sql.DB, code rune) ([]Ingredient, error) {
	var ingredients []Ingredient
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
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println("Everything is ok!")
		}
	}(rows)

	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.Type, &ing.Value, &ing.Price); err != nil {
			return nil, err
		}
		ingredients = append(ingredients, ing)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ingredients, nil
}

// Main function to generate dish combinations based on input codes.
func generateDishCombinations(db *sql.DB, codes string) ([]Product, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Preparing ingredient sets for each code
	ingredientSets := [][]Ingredient{}
	for _, code := range codes {
		ingredients, err := getIngredientsByCode(db, code)
		if err != nil {
			return nil, err
		}
		ingredientSets = append(ingredientSets, ingredients)
	}

	var allCombinations []Product
	combine(Product{}, ingredientSets, 0, make(map[string]bool), &allCombinations)

	return allCombinations, nil
}

// Helper function to check if a slice of ingredients already contains an ingredient by value.
func containsIngredient(ingredients []Ingredient, value string) bool {
	for _, ing := range ingredients {
		if ing.Value == value {
			return true
		}
	}
	return false
}

func main() {
	// Connect to the database
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/test_task")
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		return
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("Everything is ok!")
		}
	}(db)

	http.HandleFunc("/constructor/", func(w http.ResponseWriter, r *http.Request) {
		dishConstructorHandler(w, r, db)
	})
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Eroor starting server %s\n", err)
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

func combine(current Product, sets [][]Ingredient, index int, used map[string]bool, allCombinations *[]Product) {
	if index == len(sets) {
		// At this point, 'current' is a complete product combination
		// Add a deep copy of 'current' to the allCombinations slice to avoid reference issues
		newProd := Product{Ingredients: make([]Ingredient, len(current.Ingredients)), Price: current.Price}
		copy(newProd.Ingredients, current.Ingredients)
		*allCombinations = append(*allCombinations, newProd)
		return
	}

	for _, ing := range sets[index] {
		// Check if the ingredient's title has been used already
		if !used[ing.Value] { // Ensure unique title
			used[ing.Value] = true // Mark the ingredient's title as used
			newProduct := Product{
				Ingredients: append([]Ingredient{}, current.Ingredients...), // Create a new slice with existing ingredients
				Price:       current.Price + ing.Price,                      // Update the total price
			}
			newProduct.Ingredients = append(newProduct.Ingredients, ing) // Add the new ingredient
			// Recurse to combine the next set of ingredients
			combine(newProduct, sets, index+1, used, allCombinations)
			used[ing.Value] = false // Unmark the ingredient's title for other combinations
		}
	}
}
