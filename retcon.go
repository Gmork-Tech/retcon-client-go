package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	kjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"strings"
	"sync"
	"time"
)

type ConfigProp struct {
	id       uint32
	name     string
	priority uint16
	kind     string
	nullable bool
	val      any
}

type Props struct {
	objects    sync.Map
	slices     sync.Map
	strings    sync.Map
	booleans   sync.Map
	timestamps sync.Map
	numbers    sync.Map
	durations  sync.Map
}

type Retcon struct {
	scheme string
	host   string
	path   string
	config koanf.Koanf
	props  Props
	wg     *sync.WaitGroup
}

func (rc *Retcon) getBoolean(name string) *bool {
	values, ok := rc.props.booleans.Load(name)
	if !ok || values == nil {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*bool)
}

func (rc *Retcon) getString(name string) *string {
	values, ok := rc.props.strings.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*string)
}

func (rc *Retcon) getTimestamp(name string) *time.Time {
	values, ok := rc.props.timestamps.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*time.Time)
}

func (rc *Retcon) getDuration(name string) *time.Duration {
	values, ok := rc.props.durations.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*time.Duration)
}

func (rc *Retcon) getNumber(name string) *json.Number {
	values, ok := rc.props.numbers.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*json.Number)
}

func (rc *Retcon) getObject(name string) *any {
	values, ok := rc.props.objects.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*any)
}

func (rc *Retcon) getSlice(name string) *[]any {
	values, ok := rc.props.slices.Load(name)
	if !ok {
		return nil
	}
	typedValues, typeOk := values.([]ConfigProp)
	if !typeOk || len(typedValues) == 0 {
		return nil
	}
	return typedValues[0].val.(*[]any)
}

func fromName(name string) *Retcon {
	var config = koanf.New(".")
	config.Load(file.Provider(name), yaml.Parser())
	config.Load(file.Provider(name), toml.Parser())
	config.Load(file.Provider(name), kjson.Parser())
	config.Load(env.Provider(strings.ToUpper(name), ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, strings.ToUpper(name))), "_", ".", -1)
	}), nil)
	return fromConfig(config)
}

func fromConfig(config *koanf.Koanf) *Retcon {
	var wg sync.WaitGroup
	retcon := Retcon{
		scheme: "ws",
		host:   config.String("host"),
		path:   "/ws/" + config.String("appId"),
		wg:     &wg,
	}
	configureClient(&wg, &retcon)
	// Spin up a new routine to handle the blocking wait group
	go retcon.wg.Wait()
	return &retcon
}

func configureClient(wg *sync.WaitGroup, retcon *Retcon) {
	conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), retcon.scheme+"://"+retcon.host+retcon.path)
	defer wg.Done()
	if err != nil {
		fmt.Printf("can not connect: %v\n", err)
	} else {
		fmt.Println("connected")
		msg := []byte("OK+OK")
		err = wsutil.WriteClientMessage(conn, ws.OpText, msg)
		if err != nil {
			fmt.Printf("can not send: %v\n", err)
			return
		} else {
			fmt.Printf("send: %s, type: %v\n", msg, ws.OpText)
		}

		msg, op, err := wsutil.ReadServerData(conn)
		if err != nil {
			fmt.Printf("can not receive: %v\n", err)
			return
		} else {
			fmt.Printf("receive: %s，type: %v\n", msg, op)
		}

		err = conn.Close()
		if err != nil {
			fmt.Printf("can not close: %v\n", err)
		} else {
			fmt.Println("closed")
		}
	}
}

func main() {
	var rc = fromName("test")
	fmt.Println(rc.path)
	time.Sleep(60 * time.Second)
}
