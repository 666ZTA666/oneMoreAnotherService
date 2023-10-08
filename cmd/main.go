package main

import (
	"log"

	"github.com/valyala/fasthttp"
	"myProject/internal/app"
)

func main() {
	// максимально не запаристо сервак с одной ручкой запускаем.
	err := fasthttp.ListenAndServe("localhost:8080", app.NewRateLimiter(app.NewService()))
	if err != nil {
		log.Fatal(err)
	}
}
