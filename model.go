package model

import (
	"errors"
	"fmt"
	"image/color"
	"io"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/qmuntal/go3mf/mesh"
)

// Identifier defines an object than can be uniquely identified.
type Identifier interface {
	Identify() (uint64, string)
}

// Object defines a composable object.
type Object interface {
	MergeToMesh(*mesh.Mesh, mgl32.Mat4)
	IsValid() bool
	IsValidForSlices(mgl32.Mat4) bool
	Type() ObjectType
}

// A Model is an in memory representation of the 3MF file.
type Model struct {
	Path                  string
	RootPath              string
	Language              string
	UUID                  string
	Units                 Units
	Thumbnail             *Attachment
	Metadata              []Metadata
	Resources             []Identifier
	BuildItems            []*BuildItem
	Attachments           []*Attachment
	ProductionAttachments []*Attachment
}

// SetThumbnail sets the package thumbnail.
func (m *Model) SetThumbnail(r io.Reader) *Attachment {
	m.Thumbnail = &Attachment{Stream: r, Path: thumbnailPath, RelationshipType: "http://schemas.openxmlformats.org/package/2006/relationships/metadata/thumbnail"}
	return m.Thumbnail
}

// MergeToMesh merges the build with the mesh.
func (m *Model) MergeToMesh(msh *mesh.Mesh) {
	for _, b := range m.BuildItems {
		b.MergeToMesh(msh)
	}
}

// FindResource returns the resource with the target unique ID.
func (m *Model) FindResource(id uint64, path string) (i Identifier, ok bool) {
	for _, value := range m.Resources {
		cid, cpath := value.Identify()
		if cid == id && cpath == path {
			i = value
			ok = true
			break
		}
	}
	return
}

// BaseMaterial defines the Model Base Material Resource.
// A model material resource is an in memory representation of the 3MF
// material resource object.
type BaseMaterial struct {
	Name  string
	Color color.RGBA
}

// ColorString returns the color as a hex string with the format #rrggbbaa.
func (m *BaseMaterial) ColorString() string {
	return fmt.Sprintf("#%x%x%x%x", m.Color.R, m.Color.G, m.Color.B, m.Color.A)
}

// BaseMaterialsResource defines a slice of BaseMaterial.
type BaseMaterialsResource struct {
	ID        uint64
	ModelPath string
	Materials []BaseMaterial
}

// Identify returns the resource ID and the ModelPath.
func (ms *BaseMaterialsResource) Identify() (uint64, string) {
	return ms.ID, ms.ModelPath
}

// Merge appends all the other base materials.
func (ms *BaseMaterialsResource) Merge(other []BaseMaterial) {
	for _, m := range other {
		ms.Materials = append(ms.Materials, BaseMaterial{m.Name, m.Color})
	}
}

// A BuildItem is an in memory representation of the 3MF build item.
type BuildItem struct {
	Object     Object
	Transform  mgl32.Mat4
	PartNumber string
	Path       string
	UUID       string
}

// HasTransform returns true if the transform is different than the identity.
func (b *BuildItem) HasTransform() bool {
	return !b.Transform.ApproxEqual(identityTransform)
}

// IsValidForSlices checks if the build object is valid to be used with slices.
func (b *BuildItem) IsValidForSlices() bool {
	return b.Object.IsValidForSlices(b.Transform)
}

// MergeToMesh merges the build object with the mesh.
func (b *BuildItem) MergeToMesh(m *mesh.Mesh) {
	b.Object.MergeToMesh(m, b.Transform)
}

// An ObjectResource is an in memory representation of the 3MF model object.
type ObjectResource struct {
	ID              uint64
	ModelPath       string
	UUID            string
	Name            string
	PartNumber      string
	SliceStackID    uint64
	SliceResoultion SliceResolution
	Thumbnail       string
	DefaultProperty interface{}
	ObjectType      ObjectType
}

// Identify returns the resource ID and the ModelPath.
func (o *ObjectResource) Identify() (uint64, string) {
	return o.ID, o.ModelPath
}

// Type returns the type of the object.
func (o *ObjectResource) Type() ObjectType {
	return o.ObjectType
}

// MergeToMesh left on purpose empty to be redefined in embedding class.
func (o *ObjectResource) MergeToMesh(m *mesh.Mesh, transform mgl32.Mat4) {
}

// IsValid should be redefined in embedding class.
func (o *ObjectResource) IsValid() bool {
	return false
}

// IsValidForSlices should be redefined in embedding class.
func (o *ObjectResource) IsValidForSlices(transform mgl32.Mat4) bool {
	return false
}

// A Component is an in memory representation of the 3MF component.
type Component struct {
	Object    Object
	Transform mgl32.Mat4
	UUID      string
}

// HasTransform returns true if the transform is different than the identity.
func (c *Component) HasTransform() bool {
	return !c.Transform.ApproxEqual(identityTransform)
}

// MergeToMesh merges a mesh with the component.
func (c *Component) MergeToMesh(m *mesh.Mesh, transform mgl32.Mat4) {
	c.Object.MergeToMesh(m, c.Transform.Mul4(transform))
}

// A ComponentResource resource is an in memory representation of the 3MF component object.
type ComponentResource struct {
	ObjectResource
	Components []*Component
}

// MergeToMesh merges the mesh with all the components.
func (c *ComponentResource) MergeToMesh(m *mesh.Mesh, transform mgl32.Mat4) {
	for _, comp := range c.Components {
		comp.MergeToMesh(m, transform)
	}
}

// IsValid checks if the component resource and all its child are valid.
func (c *ComponentResource) IsValid() bool {
	if len(c.Components) == 0 {
		return false
	}

	for _, comp := range c.Components {
		if !comp.Object.IsValid() {
			return false
		}
	}
	return true
}

// IsValidForSlices checks if the component resource and all its child are valid to be used with slices.
func (c *ComponentResource) IsValidForSlices(transform mgl32.Mat4) bool {
	if len(c.Components) == 0 {
		return true
	}

	for _, comp := range c.Components {
		if !comp.Object.IsValidForSlices(transform.Mul4(comp.Transform)) {
			return false
		}
	}
	return true
}

// A MeshResource is an in memory representation of the 3MF mesh object.
type MeshResource struct {
	ObjectResource
	Mesh                  *mesh.Mesh
	BeamLatticeAttributes BeamLatticeAttributes
}

// MergeToMesh merges the resource with the mesh.
func (c *MeshResource) MergeToMesh(m *mesh.Mesh, transform mgl32.Mat4) {
	c.Mesh.Merge(m, transform)
}

// IsValid checks if the mesh resource are valid.
func (c *MeshResource) IsValid() bool {
	if c.Mesh == nil {
		return false
	}
	switch c.ObjectType {
	case ObjectTypeModel:
		return c.Mesh.IsManifoldAndOriented()
	case ObjectTypeSupport:
		return len(c.Mesh.Beams) == 0
	case ObjectTypeSolidSupport:
		return c.Mesh.IsManifoldAndOriented()
	case ObjectTypeSurface:
		return len(c.Mesh.Beams) == 0
	}

	return false
}

// IsValidForSlices checks if the mesh resource are valid for slices.
func (c *MeshResource) IsValidForSlices(t mgl32.Mat4) bool {
	return c.SliceStackID == 0 || t[2] == 0 && t[6] == 0 && t[8] == 0 && t[9] == 0 && t[10] == 1
}

// Slice defines the resource object for slices.
type Slice struct {
	Vertices []mgl32.Vec2
	Polygons [][]int
	TopZ     float32
}

// BeginPolygon adds a new polygon and return its index.
func (s *Slice) BeginPolygon() int {
	s.Polygons = append(s.Polygons, make([]int, 0))
	return len(s.Polygons) - 1
}

// AddVertex adds a new vertex to the slice and returns its index.
func (s *Slice) AddVertex(x, y float32) int {
	s.Vertices = append(s.Vertices, mgl32.Vec2{x, y})
	return len(s.Vertices) - 1
}

// AddPolygonIndex adds a new index to the polygon.
func (s *Slice) AddPolygonIndex(polygonIndex, index int) error {
	if polygonIndex >= len(s.Polygons) {
		return errors.New("go3mf: invalid polygon index")
	}

	if index >= len(s.Vertices) {
		return errors.New("go3mf: invalid slice segment index")
	}

	p := s.Polygons[polygonIndex]
	if len(p) > 0 && p[len(p)-1] == index {
		return errors.New("go3mf: duplicated slice segment index")
	}
	s.Polygons[polygonIndex] = append(s.Polygons[polygonIndex], index)
	return nil
}

// AllPolygonsAreClosed returns true if all the polygons are closed.
func (s *Slice) AllPolygonsAreClosed() bool {
	for _, p := range s.Polygons {
		if len(p) > 1 && p[0] != p[len(p)-1] {
			return false
		}
	}
	return true
}

// IsPolygonValid returns true if the polygon is valid.
func (s *Slice) IsPolygonValid(index int) bool {
	if index >= len(s.Polygons) {
		return false
	}
	p := s.Polygons[index]
	return len(p) > 2
}

// SliceStack defines an stack of slices
type SliceStack struct {
	BottomZ      float32
	Slices       []*Slice
	UsesSliceRef bool
}

// AddSlice adds an slice to the stack and returns its index.
func (s *SliceStack) AddSlice(slice *Slice) (int, error) {
	if slice.TopZ < s.BottomZ || (len(s.Slices) != 0 && slice.TopZ < s.Slices[0].TopZ) {
		return 0, errors.New("go3mf: The z-coordinates of slices within a slicestack are not increasing")
	}
	s.Slices = append(s.Slices, slice)
	return len(s.Slices) - 1, nil
}

// SliceStackResource defines a slice stack resource.
type SliceStackResource struct {
	*SliceStack
	ID           uint64
	ModelPath    string
	TimesRefered int
}

// Identify returns the resource ID and the ModelPath.
func (s *SliceStackResource) Identify() (uint64, string) {
	return s.ID, s.ModelPath
}

// Texture2DResource Resource defines the Model Texture 2D.
type Texture2DResource struct {
	ID          uint64
	ModelPath   string
	Path        string
	ContentType Texture2DType
	TileStyleU  TileStyle
	TileStyleV  TileStyle
	Filter      TextureFilter
}

// NewTexture2DResource returns a new texture 2D resource.
func NewTexture2DResource(id uint64) *Texture2DResource {
	return &Texture2DResource{
		ID:          id,
		ContentType: PNGTexture,
		TileStyleU:  TileWrap,
		TileStyleV:  TileWrap,
		Filter:      TextureFilterAuto,
	}
}

// Identify returns the resource ID and the ModelPath.
func (t *Texture2DResource) Identify() (uint64, string) {
	return t.ID, t.ModelPath
}

// Copy copies the properties from another texture.
func (t *Texture2DResource) Copy(other *Texture2DResource) {
	t.Path = other.Path
	t.ContentType = other.ContentType
	t.TileStyleU = other.TileStyleU
	t.TileStyleV = other.TileStyleV
}