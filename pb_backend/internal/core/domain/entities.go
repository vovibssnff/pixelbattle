package domain

type Color [3]uint

type Pixel struct {
	X       uint   `json: "x"`
	Y       uint   `json: "y"`
	Color   []uint `json: "color"`
	Userid  int    `json: "userid"`
	Faculty string `json: "faculty"`
}

type RedisPixel struct {
	UserId    int    `json: "userid"`
	Faculty   string `json: "faculty"`
	Color     []uint `json: "color"`
	Timestamp int64  `json: "timestamp"`
}

type HeatMapUnit struct {
	X   uint
	Y   uint
	Len uint
}

type User struct {
	ID          int    `json: "id"`
	FirstName   string `json: "name"`
	LastName    string `json: "surname"`
	AccessToken string `json: "token"`
	Faculty     string `json: "faculty"`
}

type Image struct {
	Height uint
	Width  uint
	Data   []Pixel
}
