package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The app must work with no internet (it can run on an isolated network), so
// the built frontend may not reference anything outside the embedded files.
// Skipped until `make web` has produced dist/index.html.
func dist(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(FS, "dist")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		t.Skip("web/dist not built; run `make web`")
	}
	return sub
}

var (
	attrRe  = regexp.MustCompile(`(?:src|href)="([^"]+)"`)
	assetRe = regexp.MustCompile(`"(/[A-Za-z0-9_.-]+\.(?:svg|png|ico|jpe?g|webp|gif|woff2?|ttf|otf|css|js|json|webmanifest))"`)
	// Hosts that would mean a runtime fetch of a font, script or stylesheet.
	remoteRe = regexp.MustCompile(`fonts\.(?:googleapis|gstatic)\.com|cdn\.jsdelivr|unpkg\.com|cdnjs\.|use\.typekit|@import\s+url\(\s*['"]?https?:`)
)

func exists(t *testing.T, root fs.FS, ref string) {
	t.Helper()
	if _, err := fs.Stat(root, strings.TrimPrefix(ref, "/")); err != nil {
		t.Errorf("%q is referenced but not in the embedded dist", ref)
	}
}

func TestIndexOnlyReferencesEmbeddedFiles(t *testing.T) {
	root := dist(t)
	b, _ := fs.ReadFile(root, "index.html")
	refs := attrRe.FindAllStringSubmatch(string(b), -1)
	if len(refs) == 0 {
		t.Fatal("index.html references no assets")
	}
	for _, m := range refs {
		ref := m[1]
		if !strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "//") {
			t.Errorf("index.html reference %q is not a local absolute path", ref)
			continue
		}
		exists(t, root, ref)
	}
}

func TestBundlesHaveNoRemoteResourcesAndAssetsExist(t *testing.T) {
	root := dist(t)
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".css") || strings.HasSuffix(p, ".html")) {
			return err
		}
		b, _ := fs.ReadFile(root, p)
		if m := remoteRe.Find(b); m != nil {
			t.Errorf("%s loads a remote resource: %q", p, m)
		}
		for _, m := range assetRe.FindAllStringSubmatch(string(b), -1) {
			exists(t, root, m[1])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
