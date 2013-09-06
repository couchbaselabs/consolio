package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/dustin/go-coap"
	"github.com/golang/glog"

	"github.com/couchbaselabs/consolio/types"
)

type wireEvent struct {
	DB   string
	Type string
	Data *json.RawMessage
}

var NotFound = errors.New("not found")

var evListen = flag.String("evin", ":8675", "Event input binding")

func evInfoHandler(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	if db == nil {
		glog.Warning("db isn't ready")
		return nil
	}

	// Verify we got something sensible
	cType, ok := m.Option(coap.ContentType).(uint32)
	if !ok || coap.MediaType(cType) != coap.AppJSON {
		glog.Warningf("Got invalid content type on request: %v",
			coap.MediaType(cType))
		return nil
	}

	we := wireEvent{}
	err := json.Unmarshal(m.Payload, &we)
	if err != nil {
		glog.Warningf("Error unmarshaling %s: %v", m.Payload, err)
		return nil
	}

	if we.DB == db.Name {
		glog.Infof("Skipping update for my own bucket: %s", m.Payload)
		return nil
	}

	err = db.Update("db-"+we.DB, 0, func(current []byte) ([]byte, error) {
		if len(current) == 0 {
			return nil, NotFound
		}

		item := consolio.Item{}
		err := json.Unmarshal(current, &item)
		if err != nil {
			return nil, err
		}

		switch we.Type {
		case "state":
			// TODO:  Implement state tracking
			item.LastChange = time.Now().UTC()
			glog.Infof("Got state change for %q: %s", we.DB, *we.Data)
		case "stats":
			item.LastStat = time.Now().UTC()
			glog.Infof("Got stats for %q: %s", we.DB, *we.Data)
			item.Stats = we.Data
		default:
			return nil, fmt.Errorf("Unknown event type: %q", we.Type)
		}

		return json.Marshal(&item)
	})

	if err != nil {
		glog.Warningf("Error updating %q as %v: %s: %v",
			we.DB, we.Type, *we.Data, err)
		return nil
	}

	var rv *coap.Message
	if m.IsConfirmable() {
		rv = &coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Created,
			MessageID: m.MessageID,
		}
	}
	return rv
}

func eventListener() {
	if *evListen == "" {
		return
	}

	handler := coap.NewServeMux()
	handler.HandleFunc("/dbinfo", evInfoHandler)

	glog.Fatal(coap.ListenAndServe("udp", *evListen, handler))
}
