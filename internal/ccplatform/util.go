package ccplatform

import (
	"archive/tar"
	"time"
)

func WriteBytesToPackage(name string, payload []byte, tw *tar.Writer) error {
	//Make headers identical by using zero time
	var zeroTime time.Time
	tw.WriteHeader(
		&tar.Header{
			Name:       name,
			Size:       int64(len(payload)),
			ModTime:    zeroTime,
			AccessTime: zeroTime,
			ChangeTime: zeroTime,
			Mode:       0100644,
		})
	tw.Write(payload)

	return nil
}
