package main

import (
	"os"
	src "upload-service-auto/internal/dig"

	"github.com/joho/godotenv"
)

func loadEnv() {
	if err := godotenv.Load("./.env"); err != nil {
		panic(err)
	}
	if _, err := os.Stat("./.env.local"); err != nil {
		return
	}
	if err := godotenv.Overload("./.env.local"); err != nil {
		panic(err)
	}
}

func main() {
	loadEnv()

	kernel := src.NewKernel()

	app := src.NewApp(kernel)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
