// Package commonpressure works with pressure sensors speaking the lesker/gp dialect.
// It contains low-level mechanisms for dealing with serial connections and higher
// level mechanisms for providing an HTTP extraction over the device.
package commonpressure

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

const (
	termination = '\r'

	// comments for these direct from the manual
	pok        = "PROGM_OK"
	gainlim    = "GAIM_LIM" // Gain programmed at limit. Readout will be the pressure at max TS setting.
	opensens   = "OPN_SNSR" // Sensor defect, no change in programming. See Maintenance, Section 7.4.
	unplugsens = "SNSR_UNP" // Sensor unplugged, no change in programming.
	rangeerr   = "RANGE-ER" // Command error. TS must be set above 399 Torr, and system pressure must be above 399 Torr.
	invaliderr = "INVALID_" // System is calibrated and locked.
)

// errOnlyFunc is a function taking no arguments and returning nil or an error
type errOnlyFunc func() error
type strErrFunc func() (string, error)

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// Pressure is a struct holding a single variable P used for http responses
type Pressure struct {
	P float64 `json:"p"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (press *Pressure) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(press)
	if err != nil {
		fstr := fmt.Sprintf("error encoding pressure data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// Sensor has an address and connection type and can make commands
type Sensor struct {
	*comm.RemoteDevice
	server.Server
}

// NewSensor creates a new Sensor instance
func NewSensor(addr, urlStem string, serial bool) *Sensor {
	rd := comm.NewRemoteDevice(addr, serial, nil, makeSerConf(addr))
	srv := server.NewServer(urlStem)
	s := Sensor{RemoteDevice: &rd}
	srv.RouteTable["pressure"] = s.HTTPRead
	srv.RouteTable["factory-reset"] = s.HTTPFactoryReset
	srv.RouteTable["void-calibration"] = s.HTTPVoidCal
	srv.RouteTable["set-span"] = s.HTTPSetSpan
	srv.RouteTable["set-zero"] = s.HTTPSetZero
	srv.RouteTable["version"] = s.HTTPVersion
	s.Server = srv
	return &s
}

// SerialConf returns a serial config and satisfies SerialConfigurator
func (s *Sensor) SerialConf() serial.Config {
	return *makeSerConf(s.Addr)
}

// Send overloads RemoteDevice.Send to prepend the "#01" that is expected by the sensor
func (s *Sensor) Send(b []byte) error {
	o := make([]byte, 0, len(b)+3)
	o = append(o, []byte("#01")...)
	o = append(o, b...)
	return s.RemoteDevice.Send(o)
}

// Read polls the Sensor for the current temperature and humidity, opening and closing a connection along the way
func (s *Sensor) Read() (float64, error) {
	cmd := []byte("RD")
	err := s.Open()
	if err != nil {
		return 0, err
	}
	defer s.CloseEventually()
	r, err := s.SendRecv(cmd)
	if err != nil {
		return 0, err
	}
	resp := string(r)
	strs := strings.Split(resp, " ")
	f, err := strconv.ParseFloat(strs[1], 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// runSetOnlyCommand executes a command that we do not care about non-error
// responses to and returns any errors encountered along the way
func (s *Sensor) runSetOnlyCommand(cmd string) error {
	cmdB := []byte(cmd)
	err := s.Open()
	if err != nil {
		return err
	}
	defer s.CloseEventually()
	r, err := s.SendRecv(cmdB)
	if err != nil {
		return err
	}
	sr := string(r)
	if sr == pok {
		return nil
	}
	return errors.New(sr)
}

// GetVer gets the version string from the sensor
func (s *Sensor) GetVer() (string, error) {
	var ret string
	cmd := []byte("VER")
	err := s.Open()
	if err != nil {
		return ret, err
	}
	defer s.CloseEventually()
	r, err := s.SendRecv(cmd)
	if err != nil {
		return ret, err
	}
	return string(r), nil
}

// SetSpan sets the span "high point / atmosphere" of the sensor
func (s *Sensor) SetSpan() error {
	return s.runSetOnlyCommand("TS")
}

// SetZero sets the zero pressure point of the sensor
func (s *Sensor) SetZero() error {
	return s.runSetOnlyCommand("TZ")
}

// VoidCalibration voids the NIST tracable calibration
func (s *Sensor) VoidCalibration() error {
	return s.runSetOnlyCommand("VC")
}

// FactoryReset wipes away the sensor's set points and calibrations and requires a power cycle after running
func (s *Sensor) FactoryReset() error {
	return s.runSetOnlyCommand("FAC")
}

// HTTPRead handles the single route served by a Sensor
func (s *Sensor) HTTPRead(w http.ResponseWriter, r *http.Request) {
	f, err := s.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	hp := server.HumanPayload{Float: f, T: types.Float64}
	hp.EncodeAndRespond(w, r)
	return
}

func (s *Sensor) httpWriteOnly(f errOnlyFunc, w http.ResponseWriter, r *http.Request) {
	err := f()
	if err == nil {
		w.WriteHeader(200)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

func (s *Sensor) httpReturnString(f strErrFunc, w http.ResponseWriter, r *http.Request) {
	ss, err := f()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	hp := server.HumanPayload{String: ss, T: types.String}
	hp.EncodeAndRespond(w, r)
	return
}

// HTTPVoidCal reacts to a GET request by voiding the calibration and returning 200/ok,
// or any error generated by the sensor
func (s *Sensor) HTTPVoidCal(w http.ResponseWriter, r *http.Request) {
	s.httpWriteOnly(s.VoidCalibration, w, r)
	return
}

// HTTPFactoryReset reacts to a GET request by resetting the sensor to its
// factory state (requiring a power cycle thereafter) and returning 200/ok
// or any error generated by the sensor
func (s *Sensor) HTTPFactoryReset(w http.ResponseWriter, r *http.Request) {
	s.httpWriteOnly(s.FactoryReset, w, r)
	return
}

// HTTPSetSpan reacts to a GET request by setting the span of the sensor
// to the current pressure and returning 200/ok
// or any error generated by the sensor
func (s *Sensor) HTTPSetSpan(w http.ResponseWriter, r *http.Request) {
	s.httpWriteOnly(s.SetSpan, w, r)
	return
}

// HTTPSetZero reacts to a GET request by setting the zero point of the sensor
// to the current pressure and returning 200/ok
// or any error generated by the sensor
func (s *Sensor) HTTPSetZero(w http.ResponseWriter, r *http.Request) {
	s.httpWriteOnly(s.SetZero, w, r)
	return
}

// HTTPVersion reacts to a GET request by relaying the version information from
// the sensor and returning 200/ok
// or any error generated by the sensor
func (s *Sensor) HTTPVersion(w http.ResponseWriter, r *http.Request) {
	s.httpReturnString(s.GetVer, w, r)
}
