package app

import (
	"errors"
	"sync"
)

// myLocker - Интерфейс для локера с возможностью отдельно лочить чтение и запись.
type myLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// storage - хранилище для айтемов, которое будет их складывать в ведра и выдавать ведрами.
type storage struct {
	// слайс с батчами
	s []Batch
	// локер
	lock myLocker
	// количество элементов в одном батче.
	counter uint64
}

// newStorage - конструктор для хранилища батчей.
func newStorage(counter uint64) *storage {
	return &storage{counter: counter, lock: new(sync.RWMutex), s: make([]Batch, 0)}
}

// ошибка возвращаемая в случае пустого хранилища.
var errEmptyStorage = errors.New("empty storage")

// get - отдает по возможности наполненный батч и ошибку, если хранилище пустое.
func (s *storage) get() (Batch, error) {
	s.lock.RLock()
	// соответственно проверка, на пустоту хранилища.
	if len(s.s) == 0 {
		s.lock.RUnlock()
		return nil, errEmptyStorage
	}
	// вытаскиваем первый баскет
	res := s.s[0]
	s.lock.RUnlock()
	s.lock.Lock()
	// удаляем первый баскет.
	s.s = s.s[1:]
	s.lock.Unlock()
	// возвращаем первый баскет.
	return res, nil
}

func (s *storage) add(i Item) error {
	// индекс батча в который будем писать
	n := 0
	s.lock.RLock()
	switch {
	// Если хранилище пустое,
	case len(s.s) == 0:
		// то выходим из свитча, у нас уже нулевой индекс.
		break
		// Если количество элементов в крайнем из слайса батче меньше каунтера,
	case uint64(len(s.s[len(s.s)-1])) < s.counter:
		// тогда писать будем в него.
		n = len(s.s) - 1
		// если количество элементов в крайнем из слайса батче больше (хотя скорее всего только равно)
		// или равно каунтеру, тогда
	case uint64(len(s.s[len(s.s)-1])) >= s.counter:
		// писать будем в новый элемент индекс которого равен длине слайса.
		n = len(s.s)
	}
	s.lock.RUnlock()
	s.lock.Lock()
	defer s.lock.Unlock()
	// если n равно длине слайса (пустой слайс и кейс, когда надо добавить в самый конец)
	if n == len(s.s) {
		// добавляем элемент в слайс.
		s.s = append(s.s, []Item{i})
		// возвращаем пустую ошибку.
		return nil
	}
	// Добавляем элемент в нужный батч.
	s.s[n] = append(s.s[n], i)
	// возвращаем пустую ошибку.
	return nil
}
