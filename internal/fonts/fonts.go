// Package fonts discovers directly readable OpenType fonts in OS font folders.
package fonts

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// Entry identifies a named face inside a system font file or collection.
type Entry struct {
	Name, Path string
	Index      int
}

// SystemDirs lists user and system font locations for the current OS.
func SystemDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		win := os.Getenv("WINDIR")
		if win == "" {
			win = `C:\Windows`
		}
		dirs := []string{filepath.Join(win, "Fonts")}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
		}
		return dirs
	case "darwin":
		return []string{filepath.Join(home, "Library", "Fonts"), "/Library/Fonts", "/System/Library/Fonts", "/System/Library/AssetsV2/com_apple_MobileAsset_Font7", "/System/Library/AssetsV2/com_apple_MobileAsset_Font6"}
	default:
		return []string{filepath.Join(home, ".fonts"), filepath.Join(home, ".local", "share", "fonts"), "/usr/local/share/fonts", "/usr/share/fonts"}
	}
}

// Scan returns sorted, case-insensitively unique full font names.
func Scan(ctx context.Context, dirs []string) ([]Entry, error) {
	var entries []Entry
	seen := map[string]bool{}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			// Missing or unreadable optional font folders do not invalidate other folders.
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc", ".otc":
			default:
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			collection, err := sfnt.ParseCollectionReaderAt(f)
			if err != nil {
				return nil
			}
			for i := 0; i < collection.NumFonts(); i++ {
				face, err := collection.Font(i)
				if err != nil {
					continue
				}
				name, err := face.Name(nil, sfnt.NameIDFull)
				if err != nil || strings.TrimSpace(name) == "" {
					continue
				}
				key := strings.ToLower(name)
				if seen[key] {
					continue
				}
				seen[key] = true
				entries = append(entries, Entry{name, path, i})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// Resolve loads a listed full font name without substituting another face.
func Resolve(ctx context.Context, name string) (*sfnt.Font, error) {
	entries, err := Scan(ctx, SystemDirs())
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, name) {
			return Load(entry)
		}
	}
	return nil, fmt.Errorf("font %q not found; use 'imgstamp fonts' to list supported full font names", name)
}

// Load reads a face into memory so it remains usable after closing its file.
func Load(entry Entry) (*sfnt.Font, error) {
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return nil, err
	}
	collection, err := sfnt.ParseCollection(data)
	if err != nil {
		return nil, err
	}
	return collection.Font(entry.Index)
}
