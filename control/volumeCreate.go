package control

import (
	"io/ioutil"
	"net/http"

	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/fs"
	"github.com/golang/protobuf/proto"
)

func (c *Control) volumeCreate(w http.ResponseWriter, req *http.Request) {
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

	var msg wire.VolumeCreateRequest
	if err := proto.Unmarshal(buf, &msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = fs.Create(c.app.DB, msg.VolumeName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
