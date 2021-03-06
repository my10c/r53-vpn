// Copyright (c) 2015 - 2017 BadAssOps inc
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//	* Redistributions of source code must retain the above copyright
//	notice, this list of conditions and the following disclaimer.
//	* Redistributions in binary form must reproduce the above copyright
//	notice, this list of conditions and the following disclaimer in the
//	documentation and/or other materials provided with the distribution.
//	* Neither the name of the <organization> nor the
//	names of its contributors may be used to endorse or promote products
//	derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSEcw
// ARE DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
//
// File			:	main.go
//
// Description	:	The main client side
//
// Author		:	Luc Suryo <luc@badassops.com>
//
// Version		:	0.4
//
// Date			:	Jan 17, 2017
//
// History	:
// 	Date:			Author:		Info:
//	Feb 24, 2015	LIS			Beta release
//	Jan 3, 2017		LIS			Re-write from Python to Go
//	Jan 5, 2017		LIS			Added support for --profile and --debug
//	Jan 17, 2017	LIS			Convert to use the go objects with the adjusted r53cmd

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/my10c/r53-ufw/initialze"
	"github.com/my10c/r53-ufw/r53cmds"
	"github.com/my10c/r53-ufw/utils"

	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	logfile       string = "/tmp/r53-ufw-client.log" // default
	configName    string = "/route53"                // hardcoded
	credName      string = "/credentials"
	configAWSPath string = "/.aws"   // hardcoded
	profileName   string = "r53-ufw" // default
	r53TtlRec            = 300       // hardcoded
	r53RecType    string = route53.RRTypeA
	debug         bool   = false
	admin         bool   = false
)

func main() {
	// working variables
	var action string
	var resultARec bool = false
	var resultTxtRec bool = false

	// initialization
	configFile := os.Getenv("HOME") + configAWSPath + configName
	credFile := os.Getenv("HOME") + configAWSPath + credName
	initValue := initialze.InitArgs("client", profileName)
	if initValue == nil {
		fmt.Printf("-< Failed initialized the argument! Aborted >-\n")
		os.Exit(1)
	}
	clientAction := initValue[0]
	profileName := initValue[1]
	debug, _ := strconv.ParseBool(initValue[2])
	r53TxrRequired, _ := strconv.ParseBool(initValue[3])
	r53RecName := initValue[4]
	r53RecValue := initValue[5]
	configInfos := initialze.GetConfig(debug, profileName, configFile)
	zoneName := string(configInfos[0])
	zoneID := string(configInfos[1])
	myLog := string(configInfos[5])
	if string(myLog) != "" {
		initialze.InitLog(myLog)
	} else {
		initialze.InitLog(logfile)
	}
	mySess := r53cmds.New(admin, debug, credFile, r53TtlRec, profileName, zoneName, zoneID, r53RecName)

	if clientAction == "list" {
		mySess.FindRecords(r53RecName, 0)
		os.Exit(0)
	}
	if r53TxrRequired == true {
		r53RecType = route53.RRTypeTxt
		resultTxtRec = mySess.SearchRecord(route53.RRTypeTxt)
	}

	// let do some work ahead since we will need it
	resultARec = mySess.SearchRecord(route53.RRTypeA)

	// just for debug
	if mySess.Debug == true {
		utils.StdOutAndLog(fmt.Sprintf("** START DEBUG INFO : main **"))
		utils.StdOutAndLog(fmt.Sprintf("configFile        : %s", configFile))
		utils.StdOutAndLog(fmt.Sprintf("profileName       : %s", profileName))
		utils.StdOutAndLog(fmt.Sprintf("zoneName          : %s", mySess.ZoneName))
		utils.StdOutAndLog(fmt.Sprintf("zoneID            : %s", mySess.ZoneID))
		utils.StdOutAndLog(fmt.Sprintf("r53TxrRequired    : %t", r53TxrRequired))
		utils.StdOutAndLog(fmt.Sprintf("clientAction      : %s", clientAction))
		utils.StdOutAndLog(fmt.Sprintf("r53RecName        : %s", mySess.UserName))
		utils.StdOutAndLog(fmt.Sprintf("r53RecValue       : %s", r53RecValue))
		utils.StdOutAndLog(fmt.Sprintf("r53TtlRec         : %d", mySess.Ttl))
		utils.StdOutAndLog(fmt.Sprintf("mySess            : %v", mySess))
		utils.StdOutAndLog(fmt.Sprintf("iamUserName       : %s", mySess.IAMUserName))
		utils.StdOutAndLog(fmt.Sprintf("search Txt result : %t", resultTxtRec))
		utils.StdOutAndLog(fmt.Sprintf("search A result   : %t", resultARec))
		fmt.Print("Press 'Enter' to continue...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		utils.StdOutAndLog(fmt.Sprintf("** END DEBUG INFO **"))
	}

	switch clientAction {
	case "add":
		action = "Adding record"
		// Adding the A record
		if resultARec == false {
			if mySess.AddDelModRecord(r53RecValue, "add", route53.RRTypeA); false {
				utils.StdOutAndLog(fmt.Sprintf("Failed to add the the A-record: %s %s.", r53RecName, r53RecValue))
				os.Exit(1)
			}
			utils.PrintActionResult(action, r53RecName, r53RecValue, "IP")
		}
		if resultARec == true {
			utils.StdOutAndLog("The A-record already exist, check with action list to see value(s).")
			os.Exit(1)
		}
		// perm was given we need to add the TXT record
		if r53TxrRequired == true {
			if resultTxtRec == false {
				if mySess.AddDelModRecord(r53RecValue, "add", route53.RRTypeTxt); false {
					utils.StdOutAndLog(fmt.Sprintf("Failed to add the TXT-record: %s %s.", r53RecName, r53RecValue))
					os.Exit(1)
				}
			}
			if resultTxtRec == true {
				utils.StdOutAndLog("The TXT-record already exist, check with action list to see value(s).")
				os.Exit(1)
			}
			utils.PrintActionResult(action, r53RecName, r53cmds.TxtPrefix+r53RecValue, "TXT")
		}
	case "del":
		action = "Delete record"
		if resultARec == true {
			if mySess.AddDelModRecord(r53RecValue, "del", route53.RRTypeA); false {
				utils.StdOutAndLog(fmt.Sprintf("Failed to delete the A-record: %s %s.", r53RecName, r53RecValue))
				os.Exit(1)
			}
			utils.PrintActionResult(action, r53RecName, r53RecValue, "IP")
		}
		if resultARec == false {
			utils.StdOutAndLog("The record does not exist, check with action list to see value(s).")
			os.Exit(1)
		}
		// perm was given we need to delete the TXT record
		if r53TxrRequired == true {
			if resultTxtRec == true {
				if mySess.AddDelModRecord(r53RecValue, "del", route53.RRTypeTxt); false {
					utils.StdOutAndLog(fmt.Sprintf("Failed to delete the TXT-record: %s %s.", r53RecName, r53RecValue))
					os.Exit(1)
				}
			}
			if resultTxtRec == false {
				utils.StdOutAndLog("The TXT-record does not exist, check with action list to see value(s).")
				os.Exit(1)
			}
			utils.PrintActionResult(action, mySess.IAMUserName, r53cmds.TxtPrefix+r53RecValue, "TXT")
		}
	case "mod":
		action = "Modify record"
		if r53TxrRequired == false {
			if resultARec == true {
				resultModDel := mySess.AddDelModRecord(r53RecValue, "mod", route53.RRTypeA)
				if resultModDel == false {
					utils.StdOutAndLog(fmt.Sprintf("Failed to modify the A-record: %s %s.", r53RecName, r53RecValue))
					os.Exit(1)
				}
				utils.PrintActionResult(action, r53RecName, r53RecValue, "IP")
			}
			if resultARec == false {
				utils.StdOutAndLog("The A-record does not exist, check with action list to see value(s).")
				os.Exit(1)
			}
		}
		if r53TxrRequired == true {
			if resultTxtRec == true {
				resultModDel := mySess.AddDelModRecord(r53RecValue, "mod", route53.RRTypeTxt)
				if resultModDel == false {
					utils.StdOutAndLog(fmt.Sprintf("Failed to modify the TXT-record: %s %s.", r53RecName, r53RecValue))
					os.Exit(1)
				}
				utils.PrintActionResult(action, mySess.IAMUserName, r53cmds.TxtPrefix+r53RecValue, "TXT")
			}
			if resultTxtRec == false {
				utils.StdOutAndLog("The TXT-record does not exist, check with action list to see value(s).")
				os.Exit(1)
			}
		}
	}
	os.Exit(0)
}
