//go:build ignore
// +build ignore

package main
import (
"fmt"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/infrastructure/database"
"github.com/drama-generator/backend/domain/models"
)
func main() {
cfg, err := config.LoadConfig()
if err != nil { fmt.Println(err); return }
db, err := database.NewDatabase(cfg.Database)
if err != nil { fmt.Println(err); return }
var d models.Drama
db.Preload("Episodes.Storyboards").First(&d, 5)
for _, ep := range d.Episodes {
fmt.Printf("Ep %d has %d storyboards\n", ep.ID, len(ep.Storyboards))
}
}
