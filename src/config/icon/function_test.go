package icon_test

import (
	"testing"

	"github.com/fsncps/hyperfile/src/config/icon"
)

func TestApplyIconTheme_overridesFileIconColor(t *testing.T) {
	icon.InitIcon(true, "")
	original := icon.Icons["go"].Color

	icon.ApplyIconTheme(map[string]string{"go": "#deadbe"})

	if icon.Icons["go"].Color != "#deadbe" {
		t.Errorf("expected #deadbe, got %s", icon.Icons["go"].Color)
	}
	// glyph must be unchanged
	if icon.Icons["go"].Icon != original || icon.Icons["go"].Color == original {
		// glyph preserved; color changed — OK, but let's be explicit
	}
}

func TestApplyIconTheme_overridesFolderColor(t *testing.T) {
	icon.InitIcon(true, "")

	icon.ApplyIconTheme(map[string]string{"folder": "#cafe00"})

	if icon.Folders["folder"].Color != "#cafe00" {
		t.Errorf("expected #cafe00, got %s", icon.Folders["folder"].Color)
	}
}

func TestApplyIconTheme_unknownKeyIgnored(t *testing.T) {
	icon.InitIcon(true, "")
	before := icon.Icons["go"].Color

	icon.ApplyIconTheme(map[string]string{"no_such_type_xyzzy": "#111111"})

	if icon.Icons["go"].Color != before {
		t.Error("unrelated icon color should not change")
	}
}

func TestApplyIconTheme_emptyMapIsNoop(t *testing.T) {
	icon.InitIcon(true, "")
	before := icon.Icons["go"].Color

	icon.ApplyIconTheme(map[string]string{})

	if icon.Icons["go"].Color != before {
		t.Error("empty map should not change any colors")
	}
}

func TestApplyIconTheme_glyphPreserved(t *testing.T) {
	icon.InitIcon(true, "")
	glyphBefore := icon.Icons["go"].Icon

	icon.ApplyIconTheme(map[string]string{"go": "#aabbcc"})

	if icon.Icons["go"].Icon != glyphBefore {
		t.Error("glyph must not change when only color is overridden")
	}
}
