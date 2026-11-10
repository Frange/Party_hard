package main

// English text of the back button (backbtn.go).
func init() {
	if _, ok := langDataEN["← ATRÁS"]; !ok {
		langDataEN["← ATRÁS"] = "← BACK"
	}
}
