package ubuntu

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"dagger.io/dagger"

	"github.com/innoai-tech/hostsetup/pkg/hostsetup"
)

func (u *Ubuntu) Build(ctx context.Context, outDir string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	container := client.Container(
		dagger.ContainerOpts{
			Platform: dagger.Platform(fmt.Sprintf("linux/%s", u.OS.Arch)),
		}).
		From(fmt.Sprintf("ubuntu:%s", u.OS.Version))

	container = container.
		WithEnvVariable("DEBIAN_FRONTEND", "noninteractive").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends",
			"wget", "dpkg-dev", "ca-certificates",
			"software-properties-common",
			"gnupg2",
		})

	for name, src := range u.Sources {
		container = u.setupSource(client, container, name, src)
	}

	hasTempRepo := false

	// local temp repo
	for _, dep := range u.Deps {
		if dep.Url != "" {
			container = container.
				WithFile(fmt.Sprintf("/opt/temp-repo/debs/%s", path.Base(dep.Url)), client.HTTP(dep.Url))

			hasTempRepo = true
		}
	}

	if hasTempRepo {
		container = container.WithNewFile(
			"/etc/apt/sources.list.d/temp.list",
			`deb [trusted=yes] file:///opt/temp-repo ./
`,
		)

		container = container.WithExec([]string{
			"sh", "-c",
			"cd /opt/temp-repo && dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz",
		})
	}

	container = container.
		WithWorkdir("/opt/offline-repo/debs").
		WithExec([]string{"apt-get", "update"})

	for name, dep := range u.Deps {
		container = downloadDep(container, name, dep)
	}

	container = container.
		WithWorkdir("/opt/offline-repo").
		WithExec([]string{"sh", "-c", `find ./debs/partial/ -name '*.deb' -exec mv -t ./debs/ {} +`}).
		WithExec([]string{"sh", "-c", `rm -r ./debs/partial`}).
		WithExec([]string{"sh", "-c", "dpkg-scanpackages debs /dev/null | gzip -9c > Packages.gz"})

	if _, err := container.
		Directory("/opt/offline-repo").
		Export(
			ctx,
			path.Join(outDir, fmt.Sprintf("%s-%s", u.OS.Release(), u.OS.Arch)),
			dagger.DirectoryExportOpts{
				Wipe: true,
			},
		); err != nil {
		return err
	}

	return nil
}

func (u *Ubuntu) setupSource(client *dagger.Client, c *dagger.Container, name string, source hostsetup.Source) *dagger.Container {
	switch path.Ext(source.Url) {
	case ".list":
		// vendor repo
		c = c.WithFile(fmt.Sprintf("/etc/apt/sources.list.d/%s.list", name), client.HTTP(source.Url))
	case ".deb":
		// local repo
		debFilename := fmt.Sprintf("/opt/%s.deb", name)
		c = c.
			WithFile(debFilename, client.HTTP(source.Url)).
			WithExec([]string{
				"dpkg", "-i", debFilename,
			}).
			WithExec([]string{
				"sh", "-c",
				fmt.Sprintf("cp /var/%s-*/*-keyring.gpg /usr/share/keyrings/", name),
			})
	default:
		if strings.HasSuffix(source.Url, "/") {
			c = c.
				WithNewFile(
					fmt.Sprintf("/etc/apt/sources.list.d/%s.list", name),
					fmt.Sprintf(`
deb %s /
`, source.Url))
		}
	}

	if source.GPGKey != "" {
		gpgKeyFilename := fmt.Sprintf("/usr/share/keyrings/%s-keyring.gpg", name)

		c = c.
			WithFile(gpgKeyFilename, client.HTTP(source.GPGKey)).
			WithExec([]string{
				"apt-key", "add", gpgKeyFilename,
			})
	}

	return c
}

func downloadDep(c *dagger.Container, name string, dep hostsetup.Dep) *dagger.Container {
	pkg := name

	if dep.Version != "" {
		pkg += fmt.Sprintf("=$(apt-cache madison %s | grep %q | awk '{print $3}' | head -n 1)", name, dep.Version)
	}

	return c.
		WithExec([]string{
			"sh", "-c",
			fmt.Sprintf(`
apt-get install -y \
	--download-only --reinstall \
	-o Dir::Cache="/opt/offline-repo" \
	-o Dir::Cache::archives="debs" \
	%s
`, pkg),
		})
}
