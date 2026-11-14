package main

// ---------------------------------------------------------------------------
// SKATE PARK (v1.4.0, user): graffiti walls and a graffiti crew. The park has brick walls (gen2.go genSkatepark: its
// back wall and a free-standing wall between the two rails) and EVERY party there brings 1-3 painters of the street
// painter event (pe_graffiti.go: gfReset forces it here, gfEnd sends the next one sooner). The painters are plain
// guests (nobody is added) and the pieces stay on the walls for the rest of the party, like everywhere else.
// ---------------------------------------------------------------------------

const (
	skateWallX0, skateWallX1 = 400.0, 580.0 // the free-standing wall between the rails (330 and 640)
	skateWallY, skateWallH   = 372.0, 16.0
	skateGfFirst             = 15.0 // the first painter comes after skateGfFirst + rand*skateGfFirstVar s
	skateGfFirstVar          = 25.0
	skateGfNext              = 20.0 // and each next one skateGfNext + rand*skateGfNextVar s after the last
	skateGfNextVar           = 25.0
)

// skateGraffiti: the level is the skate park (campaign and random roll).
func skateGraffiti(d *LevelDef) bool {
	return d != nil && (d.Icon == "skate" || d.Title == "SKATE PARK")
}
