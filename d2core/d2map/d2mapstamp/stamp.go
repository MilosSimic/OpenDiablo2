package d2mapstamp

import (
	"math"
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2object"

	"github.com/OpenDiablo2/OpenDiablo2/d2common"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2data/d2datadict"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

// Stamp represents a pre-fabricated map stamp that can be placed on a map.
type Stamp struct {
	regionPath  string                       // The file path of the region
	levelType   d2datadict.LevelTypeRecord   // The level type id for this stamp
	levelPreset d2datadict.LevelPresetRecord // The level preset id for this stamp
	tiles       []d2dt1.Tile                 // The tiles contained on this stamp
	ds1         *d2ds1.DS1                   // The backing DS1 file for this stamp
}

// LoadStamp loads the Stamp data from file.
func LoadStamp(levelType d2enum.RegionIdType, levelPreset int, fileIndex int) *Stamp {
	stamp := &Stamp{
		levelType:   d2datadict.LevelTypes[levelType],
		levelPreset: d2datadict.LevelPresets[levelPreset],
	}

	for _, levelTypeDt1 := range stamp.levelType.Files {
		if len(levelTypeDt1) != 0 && levelTypeDt1 != "" && levelTypeDt1 != "0" {
			fileData, err := d2asset.LoadFile("/data/global/tiles/" + levelTypeDt1)
			if err != nil {
				panic(err)
			}

			dt1, _ := d2dt1.LoadDT1(fileData)

			stamp.tiles = append(stamp.tiles, dt1.Tiles...)
		}
	}

	var levelFilesToPick []string

	for _, fileRecord := range stamp.levelPreset.Files {
		if len(fileRecord) != 0 && fileRecord != "" && fileRecord != "0" {
			levelFilesToPick = append(levelFilesToPick, fileRecord)
		}
	}

	levelIndex := int(math.Round(float64(len(levelFilesToPick)-1) * rand.Float64()))
	if fileIndex >= 0 && fileIndex < len(levelFilesToPick) {
		levelIndex = fileIndex
	}

	if levelFilesToPick == nil {
		panic("no level files to pick from")
	}

	stamp.regionPath = levelFilesToPick[levelIndex]
	fileData, err := d2asset.LoadFile("/data/global/tiles/" + stamp.regionPath)

	if err != nil {
		panic(err)
	}

	stamp.ds1, _ = d2ds1.LoadDS1(fileData)

	// Update the region info for the tiles
	for rx := 0; rx < len(stamp.ds1.Tiles); rx++ {
		for x := 0; x < len(stamp.ds1.Tiles[rx]); x++ {
			stamp.ds1.Tiles[rx][x].RegionType = levelType
		}
	}

	return stamp
}

// Size returns the size of the stamp in tiles.
func (mr *Stamp) Size() d2common.Size {
	return d2common.Size{int(mr.ds1.Width), int(mr.ds1.Height)}
}

// LevelPreset returns the level preset ID.
func (mr *Stamp) LevelPreset() d2datadict.LevelPresetRecord {
	return mr.levelPreset
}

// LevelType returns the level type ID.
func (mr *Stamp) LevelType() d2datadict.LevelTypeRecord {
	return mr.levelType
}

// RegionPath returns the file path of the region.
func (mr *Stamp) RegionPath() string {
	return mr.regionPath
}

// Tile returns the tile at the given x and y tile coordinates.
func (mr *Stamp) Tile(x, y int) *d2ds1.TileRecord {
	return &mr.ds1.Tiles[y][x]
}

// TileData returns the tile data for the tile with given style, sequence and type.
func (mr *Stamp) TileData(style int32, sequence int32, tileType d2enum.TileType) *d2dt1.Tile {
	for _, tile := range mr.tiles {
		if tile.Style == style && tile.Sequence == sequence && tile.Type == int32(tileType) {
			return &tile
		}
	}

	return nil
}

// Entities spawns all entities and objects in this tile on the map.
func (mr *Stamp) Entities(tileOffsetX, tileOffsetY int) []d2interface.MapEntity {
	entities := make([]d2interface.MapEntity, 0)

	for _, object := range mr.ds1.Objects {
		if object.Type == int(d2enum.ObjectTypeCharacter) {
			monstat := d2datadict.MonStats[d2datadict.MonPresets[mr.ds1.Act][object.Id]]
			// If monstat is nil here it is a place_ type object, idk how to handle those yet.
			// (See monpreset and monplace txts for reference)
			if monstat != nil {
				// Temorary use of Lookup.
				npc := d2mapentity.CreateNPC((tileOffsetX*5)+object.X, (tileOffsetY*5)+object.Y, monstat, 0)
				npc.SetPaths(convertPaths(tileOffsetX, tileOffsetY, object.Paths))
				entities = append(entities, npc)
			}
		}

		if object.Type == int(d2enum.ObjectTypeItem) {
			// For objects the DS1 ID to objectID is hardcoded in the game
			// use the lookup table
			lookup := d2datadict.LookupObject(int(mr.ds1.Act), object.Type, object.Id)

			if lookup == nil {
				continue
			}

			objectRecord := d2datadict.Objects[lookup.ObjectsTxtId]

			if objectRecord != nil {
				entity, err := d2object.CreateObject((tileOffsetX*5)+object.X,
					(tileOffsetY*5)+object.Y, objectRecord, d2resource.PaletteUnits)

				if err != nil {
					panic(err)
				}

				entities = append(entities, entity)
			}
		}
	}

	return entities
}

func convertPaths(tileOffsetX, tileOffsetY int, paths []d2common.Path) []d2common.Path {
	result := make([]d2common.Path, len(paths))
	for i := 0; i < len(paths); i++ {
		result[i].Action = paths[i].Action
		result[i].X = paths[i].X + (tileOffsetX * 5)
		result[i].Y = paths[i].Y + (tileOffsetY * 5)
	}

	return result
}
