package main

import (
//	"fmt"
	"syscall"
	"time"
	"encoding/json"
	"net/http"
	)

var I2C_SLAVE	int = 0x0703
var I2CBusPath	string = "/dev/i2c-1"
var SensorAddr	int = 0x27

type SensorBytes []byte

func (self SensorBytes) MarshalJSON() ([]byte, error) {
        buf := make([]int, len(self))
        for i, c := range self {
                buf[i] = int(c)
        }
        return json.Marshal(buf)
}

type I2cBus struct {
	devfd		int
	devpath		string
}

func (self *I2cBus) Open() error {
	var err error
	self.devfd, err = syscall.Open(self.devpath, syscall.O_RDWR, 0777)
	return err
}

func (self I2cBus) SetAddr(addr int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(self.devfd), uintptr(I2C_SLAVE), uintptr(addr))
	return syscall.Errno(errno)
}


func (self I2cBus) Write(buf []byte) (int, error) {
	return syscall.Write(self.devfd, buf)
}


func (self I2cBus) Read(buf []byte) (int, error) {
	return syscall.Read(self.devfd, buf)
}



type HIH6130 struct {
	Temperature_C	float32
	Temperature_F	float32
	Humidity	float32
	SensorData	SensorBytes	`json:"RawSensorData"`
	Status		int
	Time		int64		`json:"SampleTime"`
	i2cAddr		int
	bus		*I2cBus
}

func (self *HIH6130) Init() {
	self.SensorData = make([]byte, 4)
	self.bus.Open()
	self.bus.SetAddr(self.i2cAddr)
}

func (self *HIH6130) Read() {
	self.bus.Write([]byte{0x00})
	time.Sleep(40 * time.Millisecond)
	self.bus.Read(self.SensorData)

	// http://www.phanderson.com/arduino/I2CCommunications.pdf
	self.Status = int(self.SensorData[0]) >> 6
	self.Humidity = float32(uint((self.SensorData[0] & 63)) << 8 + uint(self.SensorData[1])) / 16383 * 100
	self.Temperature_C = float32(uint(self.SensorData[2]) << 6 + uint(self.SensorData[3]) >> 2) / 16383 * 165 - 40
	self.Temperature_F = self.Temperature_C * 1.8 + 32
	self.Time = time.Now().Unix()
}

func (self *HIH6130) Daemon() {
	go func() {
		for {
//			fmt.Println("Daemon Running")
			self.Read()
			time.Sleep(5 * time.Second)
		}
	}()
}


func webHandler(w http.ResponseWriter, r *http.Request) {

        js, err := json.MarshalIndent(sensor, "", "    ")
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write(js)
}

var sensor *HIH6130

func main() {
	sensor = &HIH6130{ i2cAddr: SensorAddr,
                            bus: &I2cBus{devpath: I2CBusPath}}
	sensor.Init()
	sensor.Daemon()


/*	for i := 5; i > 0; i-- {
		time.Sleep(5 * time.Second)
		js, _ := json.MarshalIndent(sensor, "", "    ")
		fmt.Println(string(js))

	}
*/

	webMux := http.NewServeMux()
        webMux.HandleFunc("/", webHandler)
        http.ListenAndServe(":8000", webMux)

}
