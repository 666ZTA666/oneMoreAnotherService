package app

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

type serviceStub struct {
	n uint64
	t time.Duration
}

func newServiceStubForLimits(n uint64, t time.Duration) *serviceStub {
	return &serviceStub{n: n, t: t}
}

func (s *serviceStub) GetLimits() (uint64, time.Duration) {
	return s.n, s.t
}

func (s *serviceStub) Process(context.Context, Batch) error {
	return nil
}

func Test_rateLimiter_init(t *testing.T) {
	type fields struct {
		service serviceI
		storage storageI
		time    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   rateLimiter
	}{
		{
			name: "setter case",
			fields: fields{
				service: newServiceStubForLimits(10, time.Second*10),
				storage: nil,
				time:    0,
			},
			want: rateLimiter{
				service: newServiceStubForLimits(10, time.Second*10),
				storage: newStorage(10),
				time:    time.Second * 10,
			},
		},
		{
			name: "rewriter case",
			fields: fields{
				service: newServiceStubForLimits(10, time.Second*10),
				storage: newStorage(125),
				time:    time.Minute,
			},
			want: rateLimiter{
				service: newServiceStubForLimits(10, time.Second*10),
				storage: newStorage(10),
				time:    time.Second * 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &rateLimiter{
				service: tt.fields.service,
				storage: tt.fields.storage,
				time:    tt.fields.time,
			}
			a.init()
			if !reflect.DeepEqual(*a, tt.want) {
				t.Errorf("\ngot = %v\nwant= %v", *a, tt.want)
			}
		})
	}
}

type storageStub struct {
	counter          int
	b                Batch
	getErr1, getErr2 error
}

func newStorageStubForGet(b Batch, getErr1, getErr2 error) *storageStub {
	return &storageStub{b: b, getErr1: getErr1, getErr2: getErr2}
}

func (s *storageStub) add(Item) error {
	return nil
}

func (s *storageStub) get() (Batch, error) {
	defer func() { s.counter++ }()
	if s.counter%2 == 0 {
		return s.b, s.getErr1
	}
	return s.b, s.getErr2
}

type serviceStub1 struct {
	counter int
	err     error
}

func newServiceStubForProcess(err error) *serviceStub1 {
	return &serviceStub1{err: err}
}

func (s *serviceStub1) GetLimits() (uint64, time.Duration) {
	return 0, 0
}

func (s *serviceStub1) Process(context.Context, Batch) error {
	defer func() { s.counter++ }()
	if s.counter%2 == 0 {
		return s.err
	}
	return nil
}

func Test_rateLimiter_sendToService(t *testing.T) {
	type fields struct {
		service serviceI
		storage storageI
		time    time.Duration
	}
	tests := []struct {
		name               string
		fields             fields
		wantCounterStorage int
		wantCounterService int
	}{
		{
			name: "double err in storage case",
			fields: fields{
				service: newServiceStubForProcess(fmt.Errorf("err srvc")),
				storage: newStorageStubForGet(
					Batch{},
					fmt.Errorf("err storage 1"),
					fmt.Errorf("err storage 2")),
				time: 10,
			},
			wantCounterStorage: 2,
			wantCounterService: 0,
		},
		{
			name: "once err in storage, once err in service case",
			fields: fields{
				service: newServiceStubForProcess(fmt.Errorf("err srvc")),
				storage: newStorageStubForGet(
					Batch{},
					fmt.Errorf("err storage 1"),
					nil),
				time: 10,
			},
			wantCounterStorage: 2,
			wantCounterService: 2,
		},
		{
			name: "no err in storage, once err in service case",
			fields: fields{
				service: newServiceStubForProcess(fmt.Errorf("err srvc")),
				storage: newStorageStubForGet(
					Batch{},
					nil,
					nil),
				time: 10,
			},
			wantCounterStorage: 1,
			wantCounterService: 2,
		},
		{
			name: "no err in storage, no err in service case",
			fields: fields{
				service: newServiceStubForProcess(nil),
				storage: newStorageStubForGet(
					Batch{},
					nil,
					nil),
				time: 10,
			},
			wantCounterStorage: 1,
			wantCounterService: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &rateLimiter{
				service: tt.fields.service,
				storage: tt.fields.storage,
				time:    tt.fields.time,
			}
			a.sendToService()
			strg, ok := a.storage.(*storageStub)
			if !ok {
				t.Errorf("bad storage")
				return
			}
			if strg.counter != tt.wantCounterStorage {
				t.Errorf("storage\ngot = %v\nwant= %v\n", strg.counter, tt.wantCounterStorage)
			}
			srvc, ok := a.service.(*serviceStub1)
			if !ok {
				t.Errorf("bad service")
				return
			}
			if srvc.counter != tt.wantCounterService {
				t.Errorf("service\ngot = %v\nwant= %v\n", srvc.counter, tt.wantCounterService)
			}
		})
	}
}

type storageStub1 struct {
	counter    int
	err1, err2 error
}

func newStorageStubForAdd(err1, err2 error) *storageStub1 {
	return &storageStub1{err1: err1, err2: err2}
}

func (s *storageStub1) add(Item) error {
	defer func() { s.counter++ }()
	if s.counter%2 == 0 {
		return s.err1
	}
	return s.err2
}

func (s *storageStub1) get() (Batch, error) {
	return nil, nil
}

func Test_rateLimiter_handle(t *testing.T) {
	type fields struct {
		storage storageI
	}
	type args struct {
		ctx *fasthttp.RequestCtx
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantCode int
		wantBody string
	}{
		{
			name: "double err in storage",
			fields: fields{
				storage: newStorageStubForAdd(fmt.Errorf("err1"), fmt.Errorf("err2")),
			},
			args: args{
				ctx: &fasthttp.RequestCtx{},
			},
			wantCode: fasthttp.StatusInternalServerError,
			wantBody: fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
		},
		{
			name: "once err in storage",
			fields: fields{
				storage: newStorageStubForAdd(fmt.Errorf("err1"), nil),
			},
			args: args{
				ctx: &fasthttp.RequestCtx{},
			},
			wantCode: fasthttp.StatusOK,
			wantBody: fasthttp.StatusMessage(fasthttp.StatusOK),
		},
		{
			name: "no err in storage",
			fields: fields{
				storage: newStorageStubForAdd(nil, nil),
			},
			args: args{
				ctx: &fasthttp.RequestCtx{},
			},
			wantCode: fasthttp.StatusOK,
			wantBody: fasthttp.StatusMessage(fasthttp.StatusOK),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &rateLimiter{
				storage: tt.fields.storage,
			}
			a.handle(tt.args.ctx)
			sc := tt.args.ctx.Response.StatusCode()
			b := tt.args.ctx.Response.Body()
			if sc != tt.wantCode {
				t.Errorf("code\ngot = %v\nwant= %v", sc, tt.wantCode)
				return
			}
			if string(b) != tt.wantBody {
				t.Errorf("\ngot = %v\nwant= %v", string(b), tt.wantBody)
			}
		})
	}
}
