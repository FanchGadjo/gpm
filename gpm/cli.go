// Copyright 2019 Adrien Waksberg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gpm

import (
	"bufio"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"time"
)

// Cli contain config and wallet to use
type Cli struct {
	Config Config
	Wallet Wallet
}

// printEntries show entries with tables
func (c *Cli) printEntries(entries []Entry) {
	var otp string
	var tables map[string]*tablewriter.Table

	tables = make(map[string]*tablewriter.Table)

	for i, entry := range entries {
		if entry.OTP == "" {
			otp = ""
		} else {
			otp = "X"
		}
		if _, present := tables[entry.Group]; present == false {
			tables[entry.Group] = tablewriter.NewWriter(os.Stdout)
			tables[entry.Group].SetHeader([]string{"", "Name", "URI", "User", "OTP", "Comment"})
			tables[entry.Group].SetBorder(false)
			tables[entry.Group].SetColumnColor(
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgYellowColor},
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgWhiteColor},
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgCyanColor},
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgGreenColor},
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgWhiteColor},
				tablewriter.Colors{tablewriter.Normal, tablewriter.FgMagentaColor})
		}

		tables[entry.Group].Append([]string{strconv.Itoa(i), entry.Name, entry.URI, entry.User, otp, entry.Comment})
	}

	for group, table := range tables {
		fmt.Printf("\n%s\n\n", group)
		table.Render()
		fmt.Println("")
	}
}

// error print a message and exit)
func (c *Cli) error(msg string) {
	fmt.Printf("ERROR: %s\n", msg)
	os.Exit(2)
}

// input from the console
func (c *Cli) input(text string, defaultValue string, show bool) string {
	fmt.Print(text)

	if show == false {
		data, _ := terminal.ReadPassword(int(syscall.Stdin))
		text := string(data)
		fmt.Printf("\n")

		if text == "" {
			return defaultValue
		}
		return text
	}

	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	if input.Text() == "" {
		return defaultValue
	}
	return input.Text()
}

// selectEntry with a form
func (c *Cli) selectEntry() Entry {
	var index int

	entries := c.Wallet.SearchEntry(*PATTERN, *GROUP)
	if len(entries) == 0 {
		fmt.Println("no entry found")
		os.Exit(1)
	}

	c.printEntries(entries)
	if len(entries) == 1 {
		return entries[0]
	}

	c1 := make(chan int, 1)
	go func(max int) {
		for true {
			index, err := strconv.Atoi(c.input("Select the entry: ", "", true))
			if err == nil && index >= 0 && index+1 <= max {
				break
			}
			fmt.Println("your choice is not an integer or is out of range")
		}
		c1 <- index
	}(len(entries))

	select {
		case res := <-c1:
			index = res
		case <-time.After(30 * time.Second):
			os.Exit(1)
	}

	return entries[index]
}

// loadWallet get and unlock the wallet
func (c *Cli) loadWallet() {
	var walletName string

	passphrase := c.input("Enter the passphrase to unlock the wallet: ", "", false)

	if *WALLET == "" {
		walletName = c.Config.WalletDefault
	} else {
		walletName = *WALLET
	}

	c.Wallet = Wallet{
		Name:       walletName,
		Path:       fmt.Sprintf("%s/%s.gpm", c.Config.WalletDir, walletName),
		Passphrase: passphrase,
	}

	err := c.Wallet.Load()
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}
}

// List the entry of a wallet
func (c *Cli) listEntry() {
	c.loadWallet()
	entries := c.Wallet.SearchEntry(*PATTERN, *GROUP)
	if len(entries) == 0 {
		fmt.Println("no entry found")
		os.Exit(1)
	} else {
		c.printEntries(entries)
	}
}

// Delete an entry of a wallet
func (c *Cli) deleteEntry() {
	var entry Entry

	c.loadWallet()
	entry = c.selectEntry()
	confirm := c.input("are you sure you want to remove this entry [y/N] ?", "N", true)

	if confirm == "y" {
		err := c.Wallet.DeleteEntry(entry.ID)
		if err != nil {
			c.error(fmt.Sprintf("%s", err))
		}

		err = c.Wallet.Save()
		if err != nil {
			c.error(fmt.Sprintf("%s", err))
		}

		fmt.Println("the entry has been deleted")
	}
}

// Add a new entry in wallet
func (c *Cli) addEntry() {
	c.loadWallet()

	entry := Entry{}
	entry.GenerateID()
	entry.Name = c.input("Enter the name: ", "", true)
	entry.Group = c.input("Enter the group: ", "", true)
	entry.URI = c.input("Enter the URI: ", "", true)
	entry.User = c.input("Enter the username: ", "", true)
	if *RANDOM {
		entry.Password = RandomString(c.Config.PasswordLength,
			c.Config.PasswordLetter, c.Config.PasswordDigit, c.Config.PasswordSpecial)
	} else {
		entry.Password = c.input("Enter the new password: ", entry.Password, false)
	}
	entry.OTP = c.input("Enter the OTP key: ", "", false)
	entry.Comment = c.input("Enter a comment: ", "", true)

	err := c.Wallet.AddEntry(entry)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	err = c.Wallet.Save()
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	fmt.Println("the entry has been added")
}

// Update an entry in wallet
func (c *Cli) updateEntry() {
	c.loadWallet()

	entry := c.selectEntry()
	entry.Name = c.input("Enter the new name: ", entry.Name, true)
	entry.Group = c.input("Enter the new group: ", entry.Group, true)
	entry.URI = c.input("Enter the new URI: ", entry.URI, true)
	entry.User = c.input("Enter the new username: ", entry.User, true)
	if *RANDOM {
		entry.Password = RandomString(c.Config.PasswordLength,
			c.Config.PasswordLetter, c.Config.PasswordDigit, c.Config.PasswordSpecial)
	} else {
		entry.Password = c.input("Enter the new password: ", entry.Password, false)
	}
	entry.OTP = c.input("Enter the new OTP key: ", entry.OTP, false)
	entry.Comment = c.input("Enter a new comment: ", entry.Comment, true)

	err := c.Wallet.UpdateEntry(entry)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}
	c.Wallet.Save()
}

// Copy login and password from an entry
func (c *Cli) copyEntry() {
	c.loadWallet()
	entry := c.selectEntry()

	go func() {
		for true {
			choice := c.input("select one action: ", "", true)
			switch choice {
			case "l":
				clipboard.WriteAll(entry.User)
			case "p":
				clipboard.WriteAll(entry.Password)
			case "o":
				code, time, _ := entry.OTPCode()
				fmt.Printf("this OTP code is available for %d seconds\n", time)
				clipboard.WriteAll(code)
			case "q":
				clipboard.WriteAll("")
				os.Exit(0)
			default:
				fmt.Println("l -> copy login")
				fmt.Println("p -> copy password")
				fmt.Println("o -> copy OTP code")
				fmt.Println("q -> quit")
			}
		}
	}()

	select {
		case <-time.After(90 * time.Second):
			clipboard.WriteAll("")
			os.Exit(1)
	}
}

// Import entries from json file
func (c *Cli) ImportWallet() {
	c.loadWallet()

	_, err := os.Stat(*IMPORT)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	data, err := ioutil.ReadFile(*IMPORT)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	err = c.Wallet.Import(data)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	err = c.Wallet.Save()
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	fmt.Println("the import was successful")
}

// Export a wallet in json format
func (c *Cli) ExportWallet() {
	c.loadWallet()

	data, err := c.Wallet.Export()
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	err = ioutil.WriteFile(*EXPORT, data, 0600)
	if err != nil {
		c.error(fmt.Sprintf("%s", err))
	}

	fmt.Println("the export was successful")
}
