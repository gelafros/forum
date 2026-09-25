package handlers

import (
	"log"

	"forum/internal/db"
)

type Category struct {
	ID   int
	Name string
}

// Load Categories for posts
func LoadCategories() []Category {
	var categoryData Category

	categoryList := []Category{}
	queryResult, err := db.DB.Query(`SELECT id, name FROM categories ORDER BY id`)
	if err != nil {
		log.Println("Error in searching for Categories")
		return nil
	}

	defer queryResult.Close()

	// Loop through table records with .Next()
	for queryResult.Next() {

		err = queryResult.Scan(&categoryData.ID, &categoryData.Name)
		if err != nil {
			log.Println("Error inside Query Categories")
			return nil
		}
		categoryList = append(categoryList, categoryData) // Add Category Name in List
	}

	if err := queryResult.Err(); err != nil { // Check if the loop from .Next procedure was successfull
		return nil
	}

	return categoryList
}
