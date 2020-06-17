package sbdioi40

import (
	"errors"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

// Upload uploads the given snapshot of an application to the platform.
func (p *Platform) Upload(snap Snapshot) error {
	for _, snapFile := range snap.Files {
		f, err := os.Open(snapFile.Path)
		if err != nil {
			return fmt.Errorf("cannot upload %s: %v", snap.App.Name, err)
		}
		defer f.Close()

		image, err := images.Create(p.glance, images.CreateOpts{
			Name:            snap.App.Name + snapFile.Service.Name + "snap",
			DiskFormat:      snapFile.DiskFormat,
			ContainerFormat: snapFile.ContainerFormat,
		}).Extract()
		if err != nil {
			// TODO: remove already created images from destination platform
			return fmt.Errorf("cannot upload %s: %v", snap.App.Name, err)
		}
		if err := imagedata.Upload(p.glance, image.ID, f).ExtractErr(); err != nil {
			// TODO: remove already created images from destination platform
			return fmt.Errorf("uploading %s: %v", snap.App.Name, err)
		}

		// TODO: delete snapshot dir from local storage
	}

	return errors.New("not yet implemented")
}
