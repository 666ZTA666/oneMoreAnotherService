package app

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

// storageI - Интерфейс хранилища для айтемов
type storageI interface {
	add(i Item) error
	get() (Batch, error)
}

// serviceI defines external service that can process batches of items.
type serviceI interface {
	GetLimits() (n uint64, t time.Duration)
	Process(ctx context.Context, batch Batch) error
}

// rateLimiter - структура ограничивающая запросы к сервису.
type rateLimiter struct {
	service serviceI
	storage storageI
	time    time.Duration
}

// NewRateLimiter - конструктор для рейт лимитера.
func NewRateLimiter(service serviceI) fasthttp.RequestHandler {
	// Создаем рейт лимитер от сервиса
	s := &rateLimiter{service: service}
	// инициализируем сервис
	s.init()
	// Отдаем хендлер.
	return s.handle
}

// init - инициализирующая функция для рейт-лимитера.
func (a *rateLimiter) init() {
	// получаем с сервиса лимиты, количество запросов в батче и время обработки запроса.
	n, t := a.service.GetLimits()
	// Сохраняем у себя в лимитере время
	a.time = t
	// Создаем хранилище для айтемов и батчей.
	a.storage = newStorage(n)
	// запускаем горутину
	go func() {
		// с бесконечным циклом
		for {
			// Она будет ждать время работы сервиса
			time.Sleep(a.time)
			// и после этого отправлять батч в сервис
			a.sendToService()
		}
	}()
}

// extractFromCtx - Имитация вытаскивания айтема из запроса.
func extractFromCtx(_ *fasthttp.RequestCtx) Item {
	return Item{}
}

// handle - основная хендлер функция для получения айтемов.
func (a *rateLimiter) handle(ctx *fasthttp.RequestCtx) {
	// получаем из запроса айтем
	i := extractFromCtx(ctx)
	// Отправляем в хранилище
	err := a.storage.add(i)
	if err != nil {
		// если ошибка не пустая, логируем
		log.Println(err, " something wrong, try to save again")
		// и делаем ретрай
		err = a.storage.add(i)
		if err != nil {
			// если и ретрай неудачный, тогда всё сломалось, отдадим ошибку и статус код 500.
			ctx.Error(fasthttp.StatusMessage(fasthttp.StatusInternalServerError), fasthttp.StatusInternalServerError)
			return
		}
	}
	// Если смогли положить в хранилище, значит всё ок.
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte(fasthttp.StatusMessage(fasthttp.StatusOK)))
}

// sendToService - метод для отправки данных из хранилища в сервис.
func (a *rateLimiter) sendToService() {
	// пытаемся получить батч из хранилища
	batch, err := a.storage.get()
	// в случае ошибки идём проверять что к чему
	switch {
	// если ошибка говорит, что хранилище пустое, тогда залогируем это событие и выйдем
	case errors.Is(err, errEmptyStorage):
		log.Println(err, " storage is empty")
		return
	case err != nil:
		// Если ошибка не пустая и не про пустое хранилище, залогируем её
		log.Println(err, " something wrong, try again")
		// попробуем сделать ретрай
		batch, err = a.storage.get()
		// если и тут неудача
		if err != nil {
			// залогируем более страшную ошибку и выйдем.
			log.Println(err, " storage is unavailable")
			return
		}
	}
	// Отправляем батч в сервис
	err = a.service.Process(context.Background(), batch)
	// Пойдем проверять ошибку
	switch {
	// Если ошибка пустая
	case err == nil:
		// выйдем, потому что всё окс
		return
	// если ошибка, что сервис заблокирован (чего по логике быть не должно)
	case errors.Is(err, ErrBlocked):
		// Залогируем это событие
		log.Println(err, " service is blocked, retry right now")
		// Сделаем небольшой слип, в размере 10% от общего времени работы сервиса.
		time.Sleep(a.time / 10)
	case err != nil:
		// если ошибка не о том, что сервис заблочен, то логируем ошибку
		log.Println(err, " warning, something wrong, try to resend")
	}
	// и сделаем ретрай
	_ = a.service.Process(context.Background(), batch)
	// тут уж если не вышло то просто выйдем, но можно навязать логику и посложнее.
}
