package services

import (
	"regexp"
	"sort"
	"strings"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

var nonAlphaNumRe = regexp.MustCompile(`[^a-z0-9\s]+`)

// normalizeCharacterName converts names to a stable identity key so
// "Yoon Seo-yeon", "Seo-yeon", and "Yoon Seoyeon" can be compared reliably.
func normalizeCharacterName(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	if s == "" {
		return ""
	}

	// If model emits "Name (alias: X)", keep canonical part before alias note.
	if idx := strings.Index(s, "("); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}

	// Remove punctuation and normalize whitespace.
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = nonAlphaNumRe.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func characterTokens(key string) []string {
	if key == "" {
		return nil
	}
	return strings.Fields(key)
}

// isLikelySameCharacterName performs tolerant matching for near-duplicates.
// Examples:
// - "yoon seoyeon" vs "seo yeon" (after normalization rules)
// - "yoon seoyeon" vs "seoyeon"
func isLikelySameCharacterName(a, b string) bool {
	na := normalizeCharacterName(a)
	nb := normalizeCharacterName(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}

	ta := characterTokens(na)
	tb := characterTokens(nb)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}

	// Single-token short form matching the last token of a full name.
	if len(ta) == 1 && len(tb) >= 2 && ta[0] == tb[len(tb)-1] {
		return true
	}
	if len(tb) == 1 && len(ta) >= 2 && tb[0] == ta[len(ta)-1] {
		return true
	}

	// Multi-token overlap (order-independent) for robust dedupe.
	tokenSet := map[string]struct{}{}
	for _, t := range ta {
		tokenSet[t] = struct{}{}
	}
	matched := 0
	for _, t := range tb {
		if _, ok := tokenSet[t]; ok {
			matched++
		}
	}
	if matched >= 2 {
		return true
	}

	return false
}

func preferredCanonicalCharacter(a, b models.Character) models.Character {
	ka := normalizeCharacterName(a.Name)
	kb := normalizeCharacterName(b.Name)
	if len(ka) != len(kb) {
		if len(ka) > len(kb) {
			return a
		}
		return b
	}
	if a.ID <= b.ID {
		return a
	}
	return b
}

// mergeDuplicateCharactersByIdentity collapses near-duplicate character rows and
// rewires episode associations to one canonical character ID.
func mergeDuplicateCharactersByIdentity(db *gorm.DB, log *logger.Logger, dramaID uint) (int, error) {
	var chars []models.Character
	if err := db.Where("drama_id = ?", dramaID).Find(&chars).Error; err != nil {
		return 0, err
	}
	if len(chars) < 2 {
		return 0, nil
	}

	sort.Slice(chars, func(i, j int) bool { return chars[i].ID < chars[j].ID })

	mergedCount := 0
	for i := 0; i < len(chars); i++ {
		if chars[i].ID == 0 {
			continue
		}
		canonical := chars[i]
		var duplicateIDs []uint
		for j := i + 1; j < len(chars); j++ {
			if chars[j].ID == 0 {
				continue
			}
			if isLikelySameCharacterName(canonical.Name, chars[j].Name) {
				best := preferredCanonicalCharacter(canonical, chars[j])
				if best.ID != canonical.ID {
					duplicateIDs = append(duplicateIDs, canonical.ID)
					canonical = best
				} else {
					duplicateIDs = append(duplicateIDs, chars[j].ID)
				}
				chars[j].ID = 0 // mark processed
			}
		}
		if len(duplicateIDs) == 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, dupID := range duplicateIDs {
				if err := tx.Exec("UPDATE episode_characters SET character_id = ? WHERE character_id = ?", canonical.ID, dupID).Error; err != nil {
					return err
				}
				if err := tx.Where("id = ?", dupID).Delete(&models.Character{}).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return mergedCount, err
		}
		mergedCount += len(duplicateIDs)
		if log != nil {
			log.Infow("Merged duplicate characters by identity", "drama_id", dramaID, "canonical_id", canonical.ID, "canonical_name", canonical.Name, "merged_ids", duplicateIDs)
		}
	}

	return mergedCount, nil
}

