package main 

import (
	"fmt"
	"log"
	"book-scraping-and-question-generation/internal/storage"
	"github.com/joho/godotenv"
) 

func main(){
 if err := godotenv.Load(); err != nil {
	 log.Println("No .env file found")
 }
	
 storage.ConnectDB()

 fmt.Println("System is ready for data.")

}


