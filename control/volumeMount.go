package control

import (
	"io/ioutil"
	"net/http"

	"bazil.org/bazil/control/wire"
)

func (c *Control) volumeMount(w http.ResponseWriter, req *http.Request) {
	const reqMaxSize = 4096
	buf, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, reqMaxSize))
	if err != nil {
		// they really should export that error
		if err.Error() == "http: request body too large" {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var msg wire.VolumeMountRequest
	err = msg.Unmarshal(buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = c.app.Mount(msg.VolumeName, msg.Mountpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
