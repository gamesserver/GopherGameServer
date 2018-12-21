// Package gopher is used to start and change the core settings for the Gopher Game Server. The
// type ServerSettings contains all the parameters for changing the core settings. You can either
// pass a ServerSettings when calling Server.Start() or nil if you want to use the default server
// settings.
package gopher

import (
	"fmt"
	"github.com/hewiefreeman/GopherGameServer/actions"
	"github.com/hewiefreeman/GopherGameServer/database"
	"github.com/hewiefreeman/GopherGameServer/rooms"
	"github.com/hewiefreeman/GopherGameServer/users"
	"net/http"
	"strconv"
)

/////////// TO DOs:
///////////	- ServerCallbacks (DONE)
///////////		- Test
///////////	- SQLite Database:
///////////		- Save state on shut-down
///////////		- Error handle server start-up and callback on successful launch
///////////		- Above will be used to determine whether to save the state or not. If the server didn't start correctly, the sate should not be saved.
///////////		- Test
///////////	- Add checks for required ServerSettings
///////////	- Admin tools

// ServerSettings are the core settings for the Gopher Game Server. You must fill one of these out to customize
// the server's functionality to your liking.
type ServerSettings struct {
	ServerName     string // The server's name. Used for the server's ownership of private Rooms.
	MaxConnections int    // The maximum amount of concurrent connections the server will accept. Setting this to 0 means infinite.

	HostName  string // Server's host name. Use 'https://' for TLS connections. (ex: 'https://example.com')
	HostAlias string // Server's host alias name. Use 'https://' for TLS connections. (ex: 'https://www.example.com')
	IP        string // Server's IP address.
	Port      int    // Server's port.

	TLS         bool   // Enables TLS/SSL connections.
	CertFile    string // SSL/TLS certificate file location (starting from system's root folder).
	PrivKeyFile string // SSL/TLS private key file location (starting from system's root folder).

	OriginOnly bool // When enabled, the server declines connections made from outside the origin server (Admin logins always check origin). IMPORTANT: Enable this for web apps and LAN servers.

	MultiConnect   bool // Enables multiple connections under the same User. When enabled, will override KickDupOnLogin's functionality.
	KickDupOnLogin bool // When enabled, a logged in User will be disconnected from service when another User logs in with the same name.

	UserRoomControl   bool // Enables Users to create Rooms, invite/uninvite(AKA revoke) other Users to their owned private rooms, and destroy their owned rooms.
	RoomDeleteOnLeave bool // When enabled, Rooms created by a User will be deleted when the owner leaves. WARNING: If disabled, you must remember to at some point delete the rooms created by Users, or they will pile up endlessly!

	EnableSqlFeatures bool   // Enables the built-in SQL User authentication and friending. NOTE: It is HIGHLY recommended to use TLS over an SSL/HTTPS connection when using the SQL features. Otherwise, sensitive User information can be compromised with network "snooping" (AKA "sniffing").
	SqlIP             string // SQL Database IP address.
	SqlPort           int    // SQL Database port.
	SqlProtocol       string // The protocol to use while comminicating with the MySQL database. Most use either 'udp' or 'tcp'.
	SqlUser           string // SQL user name
	SqlPassword       string // SQL user password
	SqlDatabase       string // SQL database name
	EncryptionCost    int    // The amount of encryption iterations the server will run when storing and checking passwords. The higher the number, the longer encryptions take, but are more secure. Default is 4, range is 4-31.
	CustomLoginColumn string // The custom AccountInfoColumn you wish to use for logging in instead of the default name column.
	RememberMe        bool   // Enables the "Remember Me" login feature. You can read more about this in project's "Usage" section.

	EnableRecovery   bool   // Enables the recovery of all Rooms, their settings, and their variables on start-up after terminating the server.
	RecoveryLocation string // The folder location (starting from system's root folder) where you would like to store the recovery data.

	EnableAdminTools   bool   // Enables the use of the Admin Tools
	EnableRemoteAdmin  bool   // Enabled administration (only) from outside the origin server. When enabled, will override OriginOnly's functionality, but only for administrator connections.
	AdminToolsLogin    string // The login name for the Admin Tools
	AdminToolsPassword string // The password for the Admin Tools
}

var (
	settings  *ServerSettings
	callbacks ServerCallbacks = ServerCallbacks{}

	serverStarted bool = false

	startCallback func() = nil
	pauseCallback func() = nil
	stopCallback func() = nil
	resumeCallback func() = nil
	clientConnectCallback func(*http.ResponseWriter,*http.Request)bool = nil

	//SERVER VERSION NUMBER
	version string = "1.0-ALPHA.2"
)

// Start will start the server. Call with a pointer to your `ServerSettings` (or nil for defaults) to start the server. The default
// settings are for local testing ONLY. There are security-related options in `ServerSettings`
// for SSL/TLS, connection origin testing, administrator tools, and more. It's highly recommended to look into
// all `ServerSettings` options to tune the server for your desired functionality and security needs.
func Start(s *ServerSettings) error {
	fmt.Println(" _____             _               _____\n|  __ \\           | |             /  ___|\n| |  \\/ ___  _ __ | |__   ___ _ __\\ `--.  ___ _ ____   _____ _ __\n| | __ / _ \\| '_ \\| '_ \\ / _ \\ '__|`--. \\/ _ \\ '__\\ \\ / / _ \\ '__|\n| |_\\ \\ (_) | |_) | | | |  __/ |  /\\__/ /  __/ |   \\ V /  __/ |\n \\____/\\___/| .__/|_| |_|\\___|_|  \\____/ \\___|_|    \\_/ \\___|_|\n            | |\n            |_|                                      v" + version + "\n\n")
	fmt.Println("Starting...")
	//SET SERVER SETTINGS
	if s != nil {
		settings = s
	} else {
		//DEFAULT localhost SETTINGS
		fmt.Println("Using default settings...")
		settings = &ServerSettings{
			ServerName:     "!server!",
			MaxConnections: 0,

			HostName:  "localhost",
			HostAlias: "localhost",
			IP:        "localhost",
			Port:      8080,

			TLS:         false,
			CertFile:    "",
			PrivKeyFile: "",

			OriginOnly: false,

			MultiConnect:   false,
			KickDupOnLogin: false,

			UserRoomControl:   true,
			RoomDeleteOnLeave: true,

			EnableSqlFeatures: false,
			SqlIP:             "localhost",
			SqlPort:           3306,
			SqlProtocol:       "tcp",
			SqlUser:           "user",
			SqlPassword:       "password",
			SqlDatabase:       "database",
			EncryptionCost:    4,
			CustomLoginColumn: "",
			RememberMe:        false,

			EnableRecovery:   false,
			RecoveryLocation: "C:/",

			EnableAdminTools:   true,
			EnableRemoteAdmin:  false,
			AdminToolsLogin:    "admin",
			AdminToolsPassword: "password"}
	}

	//UPDATE SETTINGS IN users PACKAGE, THEN users WILL UPDATE SETTINGS FOR rooms PACKAGE
	users.SettingsSet((*settings).KickDupOnLogin, (*settings).ServerName, (*settings).RoomDeleteOnLeave, (*settings).EnableSqlFeatures,
		(*settings).RememberMe, (*settings).MultiConnect)

	//NOTIFY PACKAGES OF SERVER START
	serverStarted = true;
	users.SetServerStarted(true)
	rooms.SetServerStarted(true)
	actions.SetServerStarted(true)
	database.SetServerStarted(true)

	//START UP DATABASE
	if (*settings).EnableSqlFeatures {
		dbErr := database.Init((*settings).SqlUser, (*settings).SqlPassword, (*settings).SqlDatabase,
			(*settings).SqlProtocol, (*settings).SqlIP, (*settings).SqlPort, (*settings).EncryptionCost,
			(*settings).RememberMe, (*settings).CustomLoginColumn)
		if dbErr != nil {
			return dbErr
		}
		fmt.Println("Initialized Database...")
	}

	//START HTTP/SOCKET LISTENER
	if settings.TLS {
		http.HandleFunc("/wss", socketInitializer)
		if startCallback != nil {
			callbacks.Start()
		}
		err := http.ListenAndServeTLS(settings.IP+":"+strconv.Itoa(settings.Port), settings.CertFile, settings.PrivKeyFile, nil)
		if err != nil {
			return err
		}
	} else {
		http.HandleFunc("/ws", socketInitializer)
		if startCallback != nil {
			callbacks.Start()
		}
		err := http.ListenAndServe(settings.IP+":"+strconv.Itoa(settings.Port), nil)
		if err != nil {
			return err
		}
	}

	return nil
}
