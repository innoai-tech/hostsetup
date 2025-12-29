package openeuler

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"dagger.io/dagger"

	"github.com/innoai-tech/hostsetup/pkg/hostsetup"
)

func (u *OpenEuler) Build(ctx context.Context, outDir string) error {
	client, err := dagger.Connect(ctx,
		dagger.WithLogOutput(os.Stderr),
		dagger.WithVerbosity(1),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	container := client.
		Container(dagger.ContainerOpts{
			Platform: dagger.Platform(fmt.Sprintf("linux/%s", u.OS.Arch)),
		}).
		From(fmt.Sprintf("openeuler/openeuler:%s", u.OS.Version))

	container = container.
		WithExec([]string{"dnf", "makecache"}).
		WithExec([]string{"dnf", "install", "-y", "wget", "createrepo_c", "dnf-utils", "ca-certificates"})

	for name, src := range u.Sources {
		container = setupSource(client, container, name, src)
	}

	hasTempRepo := false

	for _, dep := range u.Deps {
		if dep.Url != "" && strings.HasSuffix(dep.Url, ".rpm") {
			container = container.
				WithFile(fmt.Sprintf("/opt/temp-repo/%s", path.Base(dep.Url)), client.HTTP(dep.Url))

			hasTempRepo = true
		}
	}

	if hasTempRepo {
		container = container.WithNewFile(
			"/etc/yum.repos.d/temp.repo",
			`[temp-repo]
name=Temp Local Repo
baseurl=file:///opt/temp-repo
enabled=1
gpgcheck=0
`,
		)

		container = container.WithExec([]string{"createrepo_c", "/opt/temp-repo"})
	}

	container = container.
		WithWorkdir("/opt/offline-repo/rpms").
		WithExec([]string{"dnf", "makecache"})

	for name, dep := range u.Deps {
		container = downloadDep(container, name, dep)
	}

	container = container.
		WithWorkdir("/opt/offline-repo").
		WithExec([]string{"createrepo_c", "rpms"}).
		WithNewFile("offline.repo", `[offline-repo]
name=Offline Repo
baseurl=file:///opt/offline-repo/rpms
enabled=1
gpgcheck=0
`)

	if _, err := container.
		Directory("/opt/offline-repo").
		Export(
			ctx,
			path.Join(outDir, fmt.Sprintf("%s-%s", u.OS.Release(), u.OS.Arch)),
			dagger.DirectoryExportOpts{Wipe: true},
		); err != nil {
		return err
	}

	return nil
}

func setupSource(client *dagger.Client, c *dagger.Container, name string, source hostsetup.Source) *dagger.Container {
	ext := path.Ext(source.Url)
	switch ext {
	case ".repo":
		c = c.WithFile(fmt.Sprintf("/etc/yum.repos.d/%s.repo", name), client.HTTP(source.Url))
		if source.GPGKey != "" {
			gpgPath := fmt.Sprintf("/etc/pki/rpm-gpg/RPM-GPG-KEY-%s", name)
			c = c.WithFile(gpgPath, client.HTTP(source.GPGKey)).
				WithExec([]string{"rpm", "--import", gpgPath})
		}
	case ".rpm":
		rpmFilename := fmt.Sprintf("/opt/%s.rpm", name)

		c = c.WithFile(rpmFilename, client.HTTP(source.Url)).
			WithExec([]string{"dnf", "install", "-y", rpmFilename})
	}
	return c
}

func downloadDep(c *dagger.Container, name string, dep hostsetup.Dep) *dagger.Container {
	if dep.Version != "" {
		name = fmt.Sprintf("%s-%s", name, dep.Version)
	}

	return c.
		WithExec([]string{
			"sh", "-c",
			fmt.Sprintf(`
dnf download --resolve --alldeps --destdir=/opt/offline-repo/rpms %s
`, name),
		})
}
