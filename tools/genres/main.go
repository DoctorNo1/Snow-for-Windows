// Command genres writes the Windows resources linked into the exe: the snowflake icon, a manifest that makes
// the app per-monitor DPI aware, and version information. Run it with `go generate` from the repository root.
package main

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/doctorno1/snow-for-windows/internal/icon"
	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	var images []image.Image
	for _, size := range []int{16, 20, 24, 32, 40, 48, 64, 128, 256} {
		images = append(images, icon.Draw(size))
	}
	ico, err := winres.NewIconFromImages(images)
	if err != nil {
		log.Fatal(err)
	}
	rs := &winres.ResourceSet{}
	if err := rs.SetIcon(winres.Name("APP"), ico); err != nil {
		log.Fatal(err)
	}
	rs.SetManifest(winres.AppManifest{
		Description:         "Snow for Windows",
		Compatibility:       winres.Win10AndAbove,
		ExecutionLevel:      winres.AsInvoker,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})
	vi := version.Info{FileVersion: [4]uint16{1, 0, 0, 0}, ProductVersion: [4]uint16{1, 0, 0, 0}}
	for k, v := range map[string]string{
		version.ProductName:      "Snow for Windows",
		version.FileDescription:  "Snow for Windows",
		version.ProductVersion:   "1.0.0",
		version.FileVersion:      "1.0.0",
		version.OriginalFilename: "SnowForWindows.exe",
	} {
		if err := vi.Set(0, k, v); err != nil {
			log.Fatal(err)
		}
	}
	rs.SetVersionInfo(vi)
	for name, arch := range map[string]winres.Arch{
		"rsrc_windows_amd64.syso": winres.ArchAMD64,
		"rsrc_windows_arm64.syso": winres.ArchARM64,
	} {
		f, err := os.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		if err := rs.WriteObject(f, arch); err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	// Also an .ico and a .png, handy for shortcuts and the README.
	if f, err := os.Create("assets/snowflake.ico"); err == nil {
		_ = ico.SaveICO(f)
		f.Close()
	}
	if f, err := os.Create("assets/snowflake.png"); err == nil {
		_ = png.Encode(f, icon.Draw(256))
		f.Close()
	}
}
