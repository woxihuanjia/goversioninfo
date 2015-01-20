// Copyright 2015 Joseph Spurrier
// Author: Joseph Spurrier (http://josephspurrier.com)
// License: http://www.apache.org/licenses/LICENSE-2.0.html

package goversioninfo

import (
	"math"
	"reflect"
)

// *****************************************************************************
// Structure Building
// *****************************************************************************

/*
Version Information Structures
http://msdn.microsoft.com/en-us/library/windows/desktop/ff468916.aspx

VersionInfo Names
http://msdn.microsoft.com/en-us/library/windows/desktop/aa381058.aspx#string-name

Translation: LangID
http://msdn.microsoft.com/en-us/library/windows/desktop/aa381058.aspx#langid

Translation: CharsetID
http://msdn.microsoft.com/en-us/library/windows/desktop/aa381058.aspx#charsetid

*/

// VS_VersionInfo is the top level version container.
type VS_VersionInfo struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding1     []byte
	Value        VS_FixedFileInfo
	Padding2     []byte
	Children     VS_StringFileInfo
	Children2    VS_VarFileInfo
}

// VS_FixedFileInfo - most of these should be left at the defaults.
type VS_FixedFileInfo struct {
	DwSignature        uint32
	DwStrucVersion     uint32
	DwFileVersionMS    uint32
	DwFileVersionLS    uint32
	DwProductVersionMS uint32
	DwProductVersionLS uint32
	DwFileFlagsMask    uint32
	DwFileFlags        uint32
	DwFileOS           uint32
	DwFileType         uint32
	DwFileSubtype      uint32
	DwFileDateMS       uint32
	DwFileDateLS       uint32
}

// VS_StringFileInfo holds multiple collections of keys and values,
// only allows for 1 collection in this package.
type VS_StringFileInfo struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding      []byte
	Children     VS_StringTable
}

// VS_StringTable holds a collection of string keys and values.
type VS_StringTable struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding      []byte
	Children     []VS_String
}

// VS_String holds the keys and values.
type VS_String struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding1     []byte
	Value        []byte
	Padding2     []byte
}

// VS_VarFileInfo holds the translation collection of 1.
type VS_VarFileInfo struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding      []byte
	Value        VS_Var
}

// VS_Var holds the translation key.
type VS_Var struct {
	WLength      uint16
	WValueLength uint16
	WType        uint16
	SzKey        []byte
	Padding      []byte
	Value        uint32
}

func buildString(i int, v reflect.Value) (VS_String, bool, uint16) {
	sValue := string(v.Field(i).Interface().(string))
	sName := v.Type().Field(i).Name

	ss := VS_String{}

	// If the value is set
	if sValue != "" {
		// Create key
		ss.SzKey = buildUnicode(sName, false)
		soFar := len(ss.SzKey) + 6
		ss.Padding1 = padBytes(4 - int(math.Mod(float64(soFar), 4)))
		// Ensure there is at least 4 bytes between the key and value by NOT
		// using this code
		/*if len(ss.Padding1) == 4 {
			ss.Padding1 = []byte{}
		}*/

		// Create value
		ss.Value = buildUnicode(sValue, true)
		soFar += (len(ss.Value) + len(ss.Padding1))
		ss.Padding2 = padBytes(4 - int(math.Mod(float64(soFar), 4)))
		// Eliminate too much spacing
		if len(ss.Padding2) == 4 {
			ss.Padding2 = []byte{}
		}

		// Length of text in words (2 bytes)
		ss.WValueLength = uint16(len(ss.Value) / 2)
		// This is NOT a good way because the copyright symbol counts as 2 letters
		//ss.WValueLength = uint16(len(sValue) + 1)

		// 0 for binary, 1 for text
		ss.WType = 0x01

		// Length of structure
		ss.WLength = uint16(soFar)
		// Don't include the padding in the length, but you must pass it back to
		// the parent to be included
		//ss.WLength = uint16(soFar + len(ss.Padding2))

		return ss, true, uint16(len(ss.Padding2))
	}

	return ss, false, 0
}

func buildStringTable(vi *VersionInfo) (VS_StringTable, uint16) {
	st := VS_StringTable{}

	// Always set to 0
	st.WValueLength = 0x00

	// 0 for binary, 1 for text
	st.WType = 0x01

	// Language identifier and Code page
	st.SzKey = buildUnicode(vi.VarFileInfo.Translation.getTranslationString(), false)
	soFar := len(st.SzKey) + 6
	st.Padding = padBytes(4 - int(math.Mod(float64(soFar), 4)))

	// Loop through the struct fields
	v := reflect.ValueOf(vi.StringFileInfo)
	for i := 0; i < v.NumField(); i++ {
		// If the struct is valid
		if r, ok, extra := buildString(i, v); ok {
			st.Children = append(st.Children, r)
			st.WLength += (r.WLength + extra)
		}
	}

	st.WLength += uint16(soFar)

	return st, uint16(len(st.Padding))
}

func buildStringFileInfo(vi *VersionInfo) (VS_StringFileInfo, uint16) {
	sf := VS_StringFileInfo{}

	// Always set to 0
	sf.WValueLength = 0x00

	// 0 for binary, 1 for text
	sf.WType = 0x01

	sf.SzKey = buildUnicode("StringFileInfo", false)
	soFar := len(sf.SzKey) + 6
	sf.Padding = padBytes(4 - int(math.Mod(float64(soFar), 4)))

	// Allows for more than one string table
	st, extra := buildStringTable(vi)
	sf.Children = st
	sf.WLength += (uint16(soFar) + uint16(len(sf.Padding)) + st.WLength)

	return sf, extra
}

func buildVar(vfi VarFileInfo) VS_Var {
	vs := VS_Var{}
	// Create key
	vs.SzKey = buildUnicode("Translation", false)
	soFar := len(vs.SzKey) + 6
	vs.Padding = padBytes(4 - int(math.Mod(float64(soFar), 4)))

	// Create value
	vs.Value = str2Uint32(vfi.Translation.getTranslation())
	soFar += (4 + len(vs.Padding))

	// Length of text in bytes
	vs.WValueLength = 4

	// 0 for binary, 1 for text
	vs.WType = 0x00

	// Length of structure
	vs.WLength = uint16(soFar)

	return vs
}

func buildVarFileInfo(vfi VarFileInfo) VS_VarFileInfo {
	vf := VS_VarFileInfo{}

	// Always set to 0
	vf.WValueLength = 0x00

	// 0 for binary, 1 for text
	vf.WType = 0x01

	vf.SzKey = buildUnicode("VarFileInfo", false)
	soFar := len(vf.SzKey) + 6
	vf.Padding = padBytes(4 - int(math.Mod(float64(soFar), 4)))

	// Allows for more than one string table
	st := buildVar(vfi)
	vf.Value = st
	vf.WLength += (uint16(soFar) + uint16(len(vf.Padding)) + st.WLength)

	return vf
}

func buildFixedFileInfo(vi *VersionInfo) VS_FixedFileInfo {
	ff := VS_FixedFileInfo{}
	ff.DwSignature = 0xFEEF04BD
	ff.DwStrucVersion = 0x00010000
	ff.DwFileVersionMS = str2Uint32(vi.FixedFileInfo.FileVersion.getVersionHighString())
	ff.DwFileVersionLS = str2Uint32(vi.FixedFileInfo.FileVersion.getVersionLowString())
	ff.DwProductVersionMS = str2Uint32(vi.FixedFileInfo.ProductVersion.getVersionHighString())
	ff.DwProductVersionLS = str2Uint32(vi.FixedFileInfo.ProductVersion.getVersionLowString())
	ff.DwFileFlagsMask = str2Uint32(vi.FixedFileInfo.FileFlagsMask)
	ff.DwFileFlags = str2Uint32(vi.FixedFileInfo.FileFlags)
	ff.DwFileOS = str2Uint32(vi.FixedFileInfo.FileOS)
	ff.DwFileType = str2Uint32(vi.FixedFileInfo.FileType)
	ff.DwFileSubtype = str2Uint32(vi.FixedFileInfo.FileSubType)

	// According to the spec, these should be zero...ugh
	/*if vi.Timestamp {
		now := syscall.NsecToFiletime(time.Now().UnixNano())
		ff.DwFileDateMS = now.HighDateTime
		ff.DwFileDateLS = now.LowDateTime
	}*/

	return ff
}

// Build fills the structs with data from the config file
func (v *VersionInfo) Build() {
	vi := VS_VersionInfo{}

	// 0 for binary, 1 for text
	vi.WType = 0x00

	vi.SzKey = buildUnicode("VS_VERSION_INFO", false)
	soFar := len(vi.SzKey) + 6
	vi.Padding1 = padBytes(4 - int(math.Mod(float64(soFar), 4)))

	vi.Value = buildFixedFileInfo(v)

	// Length of value (always the same)
	vi.WValueLength = 0x34

	// Never needs padding
	vi.Padding2 = []byte{}

	// Build strings
	sf, extraPadding := buildStringFileInfo(v)
	vi.Children = sf

	// Build translation
	vf := buildVarFileInfo(v.VarFileInfo)
	vi.Children2 = vf

	// Calculate the total size
	vi.WLength += (uint16(soFar) + uint16(len(vi.Padding1)) + vi.WValueLength + uint16(len(vi.Padding2)) + vi.Children.WLength + vi.Children2.WLength + extraPadding)

	v.Structure = vi
}
