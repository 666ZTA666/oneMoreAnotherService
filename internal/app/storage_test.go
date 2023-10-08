package app

import (
	"reflect"
	"testing"
)

type localRWMutex struct {
	lock  bool
	rLock bool
}

func newLocalRWMutex() *localRWMutex {
	return &localRWMutex{}
}

func (l *localRWMutex) Lock() {
	if l.lock || l.rLock {
		panic("lock")
	}
	l.lock = true
}

func (l *localRWMutex) Unlock() {
	if !l.lock || l.rLock {
		panic("unlock")
	}
	l.lock = false
}

func (l *localRWMutex) RLock() {
	if l.lock || l.rLock {
		panic("rLock")
	}
	l.rLock = true
}

func (l *localRWMutex) RUnlock() {
	if l.lock || !l.rLock {
		panic("rUnlock")
	}
	l.rLock = false
}

func Test_storage_get(t *testing.T) {
	type fields struct {
		s    []Batch
		lock myLocker
	}
	tests := []struct {
		name          string
		fields        fields
		want          Batch
		wantInStorage []Batch
		wantErr       bool
	}{
		{
			name: "nil storage",
			fields: fields{
				s:    nil,
				lock: newLocalRWMutex(),
			},
			want:          nil,
			wantInStorage: nil,
			wantErr:       true,
		},
		{
			name: "empty storage",
			fields: fields{
				s:    []Batch{},
				lock: newLocalRWMutex(),
			},
			want:          nil,
			wantInStorage: []Batch{},
			wantErr:       true,
		},
		{
			name: "1 elem case",
			fields: fields{
				s:    []Batch{{{}}},
				lock: newLocalRWMutex(),
			},
			want:          Batch{{}},
			wantInStorage: []Batch{},
			wantErr:       false,
		},
		{
			name: "2 elem case",
			fields: fields{
				s:    []Batch{{{}}, {}},
				lock: newLocalRWMutex(),
			},
			want:          Batch{{}},
			wantInStorage: []Batch{{}},
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &storage{
				s:    tt.fields.s,
				lock: tt.fields.lock,
			}
			got, err := s.get()
			if (err != nil) != tt.wantErr {
				t.Errorf("get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(s.s, tt.wantInStorage) {
				t.Errorf("get() got = %v, want %v", s.s, tt.wantInStorage)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_storage_add(t *testing.T) {
	type fields struct {
		s       []Batch
		lock    myLocker
		counter uint64
	}
	type args struct {
		i Item
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Batch
		wantErr bool
	}{
		{
			name: "nil storage case",
			fields: fields{
				s:       nil,
				lock:    newLocalRWMutex(),
				counter: 10,
			},
			args: args{
				i: Item{},
			},
			want:    []Batch{{{}}},
			wantErr: false,
		},
		{
			name: "empty storage case",
			fields: fields{
				s:       []Batch{},
				lock:    newLocalRWMutex(),
				counter: 10,
			},
			args: args{
				i: Item{},
			},
			want:    []Batch{{{}}},
			wantErr: false,
		},
		{
			name: "non empty storage case",
			fields: fields{
				s:       []Batch{{{}}},
				lock:    newLocalRWMutex(),
				counter: 10,
			},
			args: args{
				i: Item{},
			},
			want:    []Batch{{{}, {}}},
			wantErr: false,
		},
		{
			name: "first batch is full case",
			fields: fields{
				s:       []Batch{{{}, {}}},
				lock:    newLocalRWMutex(),
				counter: 2,
			},
			args: args{
				i: Item{},
			},
			want:    []Batch{{{}, {}}, {{}}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &storage{
				s:       tt.fields.s,
				lock:    tt.fields.lock,
				counter: tt.fields.counter,
			}
			if err := s.add(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(s.s, tt.want) {
				t.Errorf("\ngot = %v\nwant= %v", s.s, tt.want)
			}
		})
	}
}
