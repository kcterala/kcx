/*
Copyright © 2025 NAME HERE kcterala@gmail.com
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// nowCmd represents the now command
var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "shows time with different time zones",
	Long:  `Shows time with different timezones with AM and PM functionalities`,
	Run: func(cmd *cobra.Command, args []string) {
		printTime()
	},
}

func printTime() {
	blue := color.New(color.FgBlue).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Get the current time
    now := time.Now()
    utc := now.UTC()

    // Time in IST (Indian Standard Time)
    locationIST, _ := time.LoadLocation("Asia/Kolkata")
    ist := now.In(locationIST)

    // Time in system's local timezone
    local := now.Local()

    // 24-hour format heading
    fmt.Printf("%s:\n\n", blue("24-hour format"))
    fmt.Printf("IST   : %s\n", green(ist.Format("2006-01-02 15:04:05.000")))
    fmt.Printf("UTC   : %s\n", green(utc.Format("2006-01-02 15:04:05.000")))
    if !ist.Equal(local) {
        fmt.Printf("Local : %s\n", green(local.Format("2006-01-02 15:04:05.000")))
    }
    fmt.Println()

    // 12-hour format heading
    fmt.Printf("%s:\n\n", blue("12-hour format"))
    fmt.Printf("IST   : %s\n", green(ist.Format("2006-01-02 03:04:05.000 PM")))
    fmt.Printf("UTC   : %s\n", green(utc.Format("2006-01-02 03:04:05.000 PM")))
    if !ist.Equal(local) {
        fmt.Printf("Local : %s\n", green(local.Format("2006-01-02 03:04:05.000 PM")))
    }
    fmt.Println()
}

func init() {
	rootCmd.AddCommand(nowCmd)
}
