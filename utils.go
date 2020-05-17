package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
	"github.com/ysmood/rod/lib/proto"
)

// Array of any type
type Array []interface{}

// SprintFnApply is a helper to render template into js code
// js looks like "(a, b) => {}", the a and b are the params passed into the function
func sprintFnApply(js string, params Array) string {
	const tpl = `(%s).apply(this, %s)`

	return fmt.Sprintf(tpl, js, kit.MustToJSON(params))
}

// SprintFnThis wrap js with this
func SprintFnThis(js string) string {
	return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, js)
}

// CancelPanic graceful panic
func CancelPanic(err error) {
	if err != nil && err != context.Canceled {
		panic(err)
	}
}

// Event helps to convert a cdp.Event to proto.Event. Returns false if the conversion fails
func Event(msg *cdp.Event, evt proto.Event) bool {
	if msg.Method == evt.MethodName() {
		err := json.Unmarshal(msg.Params, evt)
		return err == nil
	}
	return false
}

// NewEventFilter creates a event filter, when matches it will load data into the event object
func NewEventFilter(event proto.Event) EventFilter {
	return func(e *cdp.Event) bool {
		if event.MethodName() == e.Method {
			kit.E(json.Unmarshal(e.Params, event))
			return true
		}
		return false
	}
}

func isNilContextErr(err error) bool {
	if err == nil {
		return false
	}
	cdpErr, ok := err.(*cdp.Error)
	return ok && cdpErr.Code == -32000
}

func matchWithFilter(s string, includes, excludes []string) bool {
	for _, include := range includes {
		if regexp.MustCompile(include).MatchString(s) {
			for _, exclude := range excludes {
				if regexp.MustCompile(exclude).MatchString(s) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func saveScreenshot(bin []byte, toFile []string) {
	if len(toFile) == 0 {
		return
	}
	if toFile[0] == "" {
		toFile = []string{"tmp", "screenshots", fmt.Sprintf("%d", time.Now().UnixNano()) + ".png"}
	}
	kit.E(kit.OutputFile(filepath.Join(toFile...), bin, nil))
}

func ginHTML(ctx kit.GinContext, body string) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	kit.E(ctx.Writer.WriteString(body))
}

func eachEvent(ob *kit.Observable, fn interface{}) {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)
	eventType := fnType.In(0).Elem()
	sub := ob.Subscribe()
	defer ob.Unsubscribe(sub)
	for e := range sub.C {
		event := reflect.New(eventType)
		if Event(e.(*cdp.Event), event.Interface().(proto.Event)) {
			ret := fnVal.Call([]reflect.Value{event})
			if len(ret) > 0 {
				if ret[0].Bool() {
					break
				}
			}
		}
	}
}
