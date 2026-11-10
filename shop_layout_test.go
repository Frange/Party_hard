package main

import (
	"fmt"
	"testing"
)

// The PUNTOS line never touches the group row (VESTUARIO's frame included) nor the TIENDA title, whatever the figure.
func TestShopPointsLineClearOfGroups(t *testing.T) {
	g := textGame()
	type rect struct{ x0, y0, x1, y1 float64 }
	hit := func(a, b rect) bool { return a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1 }
	obstacles := []rect{{244, shopTitleY, 244 + g.textW("TIENDA", 1.6), shopTitleY + shopPointsH}}
	for gi := range shopGroups {
		gx := shopGroupX(gi)
		obstacles = append(obstacles, rect{gx - 4, shopGroupFrameY, gx - 4 + shopGroupW + 2, shopGroupFrameY + 22})
	}
	for _, w := range []int{0, 5, 34555, 1234567, 99999999} {
		s := fmt.Sprintf("PUNTOS: %d", w)
		x1 := shopPointsRight
		r := rect{x1 - g.textW(s, 1.6), shopPointsY, x1, shopPointsY + shopPointsH}
		if r.x1 > 930 {
			t.Fatalf("%s sticks out of the panel", s)
		}
		for _, o := range obstacles {
			if hit(r, o) {
				t.Fatalf("%s %v overlaps %v", s, r, o)
			}
		}
	}
}
