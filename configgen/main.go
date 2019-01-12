package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type universe struct {
	Jumps        []jump `json:"jumps"`
	SolarSystems []struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Region string  `json:"region"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Z      float64 `json:"z"`
	} `json:"solarSystems"`
}

type jump struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type system struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	IPAddress string
	Prefix    string
	ASN       int
	Links     []link
}

type link struct {
	listen     bool
	port       int
	otherside  int
	dstIP      string
	internalIP int
}

type hypervisor struct {
	IP      string
	RAMLeft int
	Systems []int
}

func main() {
	b, err := ioutil.ReadFile("./universe-pretty.json")
	if err != nil {
		log.Fatalf("Unable to read JSON data file, cannot build configs without this %s", err.Error())
	}

	NewEden := universe{}
	err = json.Unmarshal(b, &NewEden)
	if err != nil {
		log.Fatalf("Unable to decode data: %s", err.Error())
	}

	systems := make(map[int]system)

	// Load every system into the map
	RAMSaved := 0
	Tick := 0
	for _, SolarSystem := range NewEden.SolarSystems {
		if SolarSystem.ID > 31000000 {
			RAMSaved += 130
			continue
		}
		Tick++
		systems[SolarSystem.ID] = system{
			ID:     SolarSystem.ID,
			Name:   SolarSystem.Name,
			Region: SolarSystem.Region,
			X:      SolarSystem.X,
			Y:      SolarSystem.Y,
			Prefix: fmt.Sprintf("2a07:1500:%x::/48", Tick),
			ASN:    Tick,
			Z:      SolarSystem.Z,
			Links:  make([]link, 0),
		}
	}

	log.Printf("Saved %dMB of ram due to wormholes", RAMSaved)

	// Dedupe the jump path map
	newJumps := make([]jump, 0)
	jumpDeDupe := make(map[string]bool)
	for _, v := range NewEden.Jumps {
		key := ""
		if v.From > v.To {
			key = fmt.Sprintf("%d-%d", v.From, v.To)
		}

		if v.From < v.To {
			key = fmt.Sprintf("%d-%d", v.To, v.From)
		}

		if !jumpDeDupe[key] {
			jumpDeDupe[key] = true
			newJumps = append(newJumps, v)
		}
	}

	NewEden.Jumps = newJumps

	// Now we need to capacity plan, We want Regions to be ont the same system at least
	// To start we need to count how much RAM is needed in every region
	RAMRegions := make(map[string]int)
	TotalRAM := 0
	for _, v := range systems {
		RAMRegions[v.Region] += 130
		TotalRAM += 130
	}

	for k, v := range RAMRegions {
		log.Printf("Region %s will require %dMB of RAM", k, v)
	}
	log.Printf("Total RAM needed %d", TotalRAM)

	hypervisors := []hypervisor{

		hypervisor{
			IP:      "127.0.0.1",
			RAMLeft: 99999999,
			Systems: make([]int, 0),
		},

		// hypervisor{
		// 	IP:      "1.1.1.1",
		// 	RAMLeft: 240000,
		// 	Systems: make([]int, 0),
		// },
		// hypervisor{
		// 	IP:      "1.1.1.2",
		// 	RAMLeft: 240000,
		// 	Systems: make([]int, 0),
		// },
		// hypervisor{
		// 	IP:      "1.1.1.3",
		// 	RAMLeft: 240000,
		// 	Systems: make([]int, 0),
		// },
	}

	// Assign systems to hypervisors
	for k, v := range RAMRegions {
		goodhvid := -1
		for hvid := 0; hvid < len(hypervisors); hvid++ {
			if hypervisors[hvid].RAMLeft > v {
				// great, there is RAM here for this region
				goodhvid = hvid
				hypervisors[hvid].RAMLeft -= v
				break
			}
		}

		if goodhvid == -1 {
			log.Fatalf("Ran out of RAM while assigning %s", k)
		}

		// Let's assign systems to this box then

		for _, sys := range systems {
			if sys.Region == k {
				// Assign time!
				hypervisors[goodhvid].Systems = append(hypervisors[goodhvid].Systems, sys.ID)

				editSystem := systems[sys.ID]
				editSystem.IPAddress = hypervisors[goodhvid].IP
				systems[sys.ID] = editSystem
			}
		}
	}

	log.Printf("Hypervisor stats %#v", hypervisors)

	// Now we loop through the links and then build some port mappings

	// to do this, I'm going to keep track of ports with a map keyed in with the HV's
	PortMap := make(map[string]int)
	for _, v := range hypervisors {
		PortMap[v.IP] = 5000
	}

	Tick = 0
	for _, jump := range NewEden.Jumps {
		SrcSystem := systems[jump.From]
		DstSystem := systems[jump.To]
		PortMap[DstSystem.IPAddress] += 2
		DstPort := PortMap[DstSystem.IPAddress]

		DstSystem.Links = append(DstSystem.Links, link{
			listen:     true,
			port:       DstPort,
			otherside:  SrcSystem.ID,
			internalIP: Tick,
		})

		SrcSystem.Links = append(SrcSystem.Links, link{
			listen:     false,
			dstIP:      DstSystem.IPAddress,
			port:       DstPort,
			otherside:  DstSystem.ID,
			internalIP: Tick + 1,
		})

		systems[jump.From] = SrcSystem
		systems[jump.To] = DstSystem
		Tick += 2
	}

	for _, v := range systems {
		log.Printf("%#v", v)
	}

	// Produce config for all systems
	for _, sys := range systems {
		dir := fmt.Sprintf("./%s/%d/", sys.IPAddress, sys.ID)
		os.MkdirAll(dir, 0777)

		interfacesConfig := "auto lo\niface lo inet loopback\n\n"

		for interfaceNumber, linkInfo := range sys.Links {
			if linkInfo.listen {
				// I need to look up the other sides prefix and linkNumber
				othersystem := systems[linkInfo.otherside]
				// linkAddress := ""
				// for otherInterfaceNumber, v := range othersystem.Links {
				// 	if !v.listen && v.dstIP == sys.IPAddress && v.port == linkInfo.port {
				// 		linkAddress = fmt.Sprintf("%s", strings.Split(othersystem.Prefix, "/")[0])
				// 	}
				// }
				interfacesConfig += fmt.Sprintf("auto eth%d\niface eth%d inet6 static\n\taddress %s%x\n\tnetmask 127\n\n",
					interfaceNumber, interfaceNumber, strings.Split(othersystem.Prefix, "/")[0], linkInfo.internalIP)

			} else {
				interfacesConfig += fmt.Sprintf("auto eth%d\niface eth%d inet6 static\n\taddress %s%x\n\tnetmask 127\n\n",
					interfaceNumber, interfaceNumber, strings.Split(sys.Prefix, "/")[0], linkInfo.internalIP)
			}
		}

		// log.Print(interfacesConfig)
		ioutil.WriteFile(dir+"interfaces", []byte(interfacesConfig), 0777)

		qemuLine := `qemu-system-i386 -kernel bzImage -hda rootfs.ext2 -append "root=/dev/sda rw" -device VGA,vgamem_mb=2 -m 64`
		qemuLine += fmt.Sprintf(" -hdb fat:./%d/", sys.ID)
		for interfaceNumber, linkInfo := range sys.Links {
			if linkInfo.listen {
				mac := quickMac()
				// -netdev socket,id=n1,mcast=239.192.168.1:1102,localaddr=1.2.3.
				// -net socket,vlan=0,udp=localhost:4444,localaddr=localhost:5555 \

				qemuLine += fmt.Sprintf(" -device e1000,netdev=eth%d,mac=52:54:00:%02x:%02x:%02x -netdev socket,id=eth%d,udp=%s:%d,localaddr=%s:%d",
					interfaceNumber, mac[0], mac[1], mac[2], interfaceNumber, linkInfo.dstIP, linkInfo.port+1, systems[linkInfo.otherside].IPAddress, linkInfo.port)

				// qemuLine += fmt.Sprintf(" -device e1000,netdev=eth%d,mac=52:54:00:%02x:%02x:%02x -netdev socket,id=eth%d,listen=:%d",
				// 	interfaceNumber, mac[0], mac[1], mac[2], interfaceNumber, linkInfo.port)
			} else {
				mac := quickMac()

				qemuLine += fmt.Sprintf(" -device e1000,netdev=eth%d,mac=52:54:00:%02x:%02x:%02x -netdev socket,id=eth%d,udp=%s:%d,localaddr=%s:%d",
					interfaceNumber, mac[0], mac[1], mac[2], interfaceNumber, linkInfo.dstIP, linkInfo.port, systems[linkInfo.otherside].IPAddress, linkInfo.port+1)

				// qemuLine += fmt.Sprintf(" -device e1000,netdev=eth%d,mac=52:54:00:%02x:%02x:%02x -netdev socket,id=eth%d,connect=%s:%d",
				// 	interfaceNumber, mac[0], mac[1], mac[2], interfaceNumber, linkInfo.dstIP, linkInfo.port)
			}
		}

		ioutil.WriteFile(dir+"qemu.sh", []byte(qemuLine), 0777)
		ioutil.WriteFile(dir+"hostname", []byte(sys.Name), 0777)

		//
		// Now build the BGP config

		birdconf := "ipv4 table master4;\nipv6 table master6;\n"

		RID := quickMac()
		birdconf += fmt.Sprintf("\nrouter id 1.%d.%d.%d;\n", RID[0], RID[1], RID[2])
		birdconf += "protocol device {\n}\n\nprotocol static announcements6{\n\tipv6;\n"
		birdconf += "\troute " + sys.Prefix + " unreachable;\n}\n\n"
		birdconf += "protocol kernel {\n\tscan time 25;\n\tipv6 {\n\t\timport none;\n\t\texport all;\n\t};\n}"

		// Now BGP peers
		for interfaceNumber, linkInfo := range sys.Links {
			if linkInfo.listen {
				othersystem := systems[linkInfo.otherside]

				birdconf += "\n\n"
				birdconf += fmt.Sprintf("protocol bgp session%d {\n", interfaceNumber)
				birdconf += fmt.Sprintf("\tneighbor %s%x as %d;\n\tsource address %s%x;\n\tlocal as %d;\n\t",
					strings.Split(othersystem.Prefix, "/")[0], linkInfo.internalIP, othersystem.ASN,
					strings.Split(othersystem.Prefix, "/")[0], linkInfo.internalIP+1, sys.ASN)

				birdconf += `
	enable extended messages;
	enable route refresh;

	ipv6 {
		import all;
		export all;
	};
}
`

				// // I need to look up the other sides prefix and linkNumber
				// othersystem := systems[linkInfo.otherside]
				// linkAddress := ""
				// for otherInterfaceNumber, v := range othersystem.Links {
				// 	if !v.listen && v.dstIP == sys.IPAddress && v.port == linkInfo.port {
				// 		linkAddress = fmt.Sprintf("%s%d", strings.Split(othersystem.Prefix, "/")[0], (otherInterfaceNumber*2)+1)
				// 	}
				// }
				// interfacesConfig += fmt.Sprintf("auto eth%d\niface eth%d inet6 static\n\taddress %s\n\tnetmask 127\n\n",
				// 	interfaceNumber, interfaceNumber, linkAddress)

			} else {
				othersystem := systems[linkInfo.otherside]

				birdconf += "\n\n"
				birdconf += fmt.Sprintf("protocol bgp session%d {\n", interfaceNumber)
				birdconf += fmt.Sprintf("\tneighbor %s%x as %d;\n\tsource address %s%x;\n\tlocal as %d;\n\t",
					strings.Split(sys.Prefix, "/")[0], linkInfo.internalIP+1, othersystem.ASN,
					strings.Split(sys.Prefix, "/")[0], linkInfo.internalIP, sys.ASN)

				birdconf += `
	enable extended messages;
	enable route refresh;

	ipv6 {
		import all;
		export all;
	};
}
`

				// interfacesConfig += fmt.Sprintf("auto eth%d\niface eth%d inet6 static\n\taddress %s%d\n\tnetmask 127\n\n",
				// 	interfaceNumber, interfaceNumber, strings.Split(sys.Prefix, "/")[0], interfaceNumber*2)
			}
		}
		ioutil.WriteFile(dir+"bird.conf", []byte(birdconf), 0777)

	}

}

func quickMac() []byte {
	o := make([]byte, 3)
	rand.Read(o)
	return o
}
