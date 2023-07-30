package cmdutils

import "github.com/pterm/pterm"

func GetENIStatusColor(status string) string {
	var stylized string
	if status == "in-use" {
		stylized = pterm.LightRed(status)
	} else {
		if status == "available" {
			stylized = pterm.LightGreen(status)
		} else {
			stylized = pterm.LightYellow(status)
		}
	}
	return stylized
}
