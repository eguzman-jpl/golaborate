package commonpressure

import (
	"go/types"
	"net/http"

	"goji.io/pat"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

func httpWriteOnly(f errOnlyFunc, w http.ResponseWriter, r *http.Request) {
	err := f()
	if err == nil {
		w.WriteHeader(200)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

func httpReturnString(f strErrFunc, w http.ResponseWriter, r *http.Request) {
	ss, err := f()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	hp := server.HumanPayload{String: ss, T: types.String}
	hp.EncodeAndRespond(w, r)
	return
}

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	Sensor

	// RouteTable maps goji patterns to http handlers
	RouteTable server.RouteTable
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(s Sensor) HTTPWrapper {
	w := HTTPWrapper{Sensor: s}
	rt := server.RouteTable{
		pat.Get("pressure"):            w.Read,
		pat.Delete("factory-reset"):    w.FactoryReset,
		pat.Delete("void-calibration"): w.VoidCal,
		pat.Post("set-span"):           w.SetSpan,
		pat.Post("set-zero"):           w.SetZero,
		pat.Get("version"):             w.Version,
	}
	w.RouteTable = rt
	return w
}

// RT satisfies server.HTTPer
func (h HTTPWrapper) RT() server.RouteTable {
	return h.RouteTable
}

// Read handles the single route served by a Sensor
func (h HTTPWrapper) Read(w http.ResponseWriter, r *http.Request) {
	f, err := h.Sensor.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	hp := server.HumanPayload{Float: f, T: types.Float64}
	hp.EncodeAndRespond(w, r)
	return
}

// VoidCal reacts to a GET request by voiding the calibration and returning 200/ok,
// or any error generated by the sensor
func (h HTTPWrapper) VoidCal(w http.ResponseWriter, r *http.Request) {
	httpWriteOnly(h.Sensor.VoidCalibration, w, r)
	return
}

// FactoryReset reacts to a GET request by resetting the sensor to its
// factory state (requiring a power cycle thereafter) and returning 200/ok
// or any error generated by the sensor
func (h HTTPWrapper) FactoryReset(w http.ResponseWriter, r *http.Request) {
	httpWriteOnly(h.Sensor.FactoryReset, w, r)
	return
}

// SetSpan reacts to a GET request by setting the span of the sensor
// to the current pressure and returning 200/ok
// or any error generated by the sensor
func (h HTTPWrapper) SetSpan(w http.ResponseWriter, r *http.Request) {
	httpWriteOnly(h.Sensor.SetSpan, w, r)
	return
}

// SetZero reacts to a GET request by setting the zero point of the sensor
// to the current pressure and returning 200/ok
// or any error generated by the sensor
func (h HTTPWrapper) SetZero(w http.ResponseWriter, r *http.Request) {
	httpWriteOnly(h.Sensor.SetZero, w, r)
	return
}

// Version reacts to a GET request by relaying the version information from
// the sensor and returning 200/ok
// or any error generated by the sensor
func (h HTTPWrapper) Version(w http.ResponseWriter, r *http.Request) {
	httpReturnString(h.Sensor.GetVer, w, r)
}
