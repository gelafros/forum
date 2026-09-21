package handlers

import (
	"forum/internal/db"
	"log"
)

type Category struct {
	ID   int
	Name string
}

// Load Categories for posts
func LoadCategories() ([]Category, error) {
	var categoryData Category

	categoryList := []Category{}
	queryResult, err := db.ForumDB.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		log.Println("Error in searching for Categories")
		return nil, err
	}

	defer queryResult.Close()

	// Loop through table records with .Next()
	for queryResult.Next() {

		err = queryResult.Scan(&categoryData.ID, &categoryData.Name)
		if err != nil {
			log.Println("Error inside Query Categories")
			return nil, err
		}
		categoryList = append(categoryList, categoryData) // Add Category Name in List
	}

	if err := queryResult.Err(); err != nil { // Check if the loop procedure was successfull
		return nil, err
	}

	return categoryList, nil
}
