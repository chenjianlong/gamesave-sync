package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/chenjianlong/gamesave-sync/pkg/gsutils"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type SearchType uint16

const (
	STKnownFolder SearchType = 1
	STRegistry    SearchType = 2
	STFolder      SearchType = 3
)

type RegistryInfo struct {
	RootKey registry.Key
	Key     string
	Name    string
}

type GameSearchInfo struct {
	Name     string     `json:"name"`
	Type     SearchType `json:"searchType"`
	FolderID *windows.KNOWNFOLDERID
	Reg      *RegistryInfo `json:"registry"`
	Dir      string        `json:"dir"`
	SubDir   string        `json:"subdir"`
	ProcName string        `json:"procName"`
}

func toKnownFolderID(folderID string) (*windows.KNOWNFOLDERID, error) {
	var knownFolderID *windows.KNOWNFOLDERID
	switch folderID {
	case "NetworkFolder":
		knownFolderID = windows.FOLDERID_NetworkFolder
	case "ComputerFolder":
		knownFolderID = windows.FOLDERID_ComputerFolder
	case "InternetFolder":
		knownFolderID = windows.FOLDERID_InternetFolder
	case "ControlPanelFolder":
		knownFolderID = windows.FOLDERID_ControlPanelFolder
	case "PrintersFolder":
		knownFolderID = windows.FOLDERID_PrintersFolder
	case "SyncManagerFolder":
		knownFolderID = windows.FOLDERID_SyncManagerFolder
	case "SyncSetupFolder":
		knownFolderID = windows.FOLDERID_SyncSetupFolder
	case "ConflictFolder":
		knownFolderID = windows.FOLDERID_ConflictFolder
	case "SyncResultsFolder":
		knownFolderID = windows.FOLDERID_SyncResultsFolder
	case "RecycleBinFolder":
		knownFolderID = windows.FOLDERID_RecycleBinFolder
	case "ConnectionsFolder":
		knownFolderID = windows.FOLDERID_ConnectionsFolder
	case "Fonts":
		knownFolderID = windows.FOLDERID_Fonts
	case "Desktop":
		knownFolderID = windows.FOLDERID_Desktop
	case "Startup":
		knownFolderID = windows.FOLDERID_Startup
	case "Programs":
		knownFolderID = windows.FOLDERID_Programs
	case "StartMenu":
		knownFolderID = windows.FOLDERID_StartMenu
	case "Recent":
		knownFolderID = windows.FOLDERID_Recent
	case "SendTo":
		knownFolderID = windows.FOLDERID_SendTo
	case "Documents":
		knownFolderID = windows.FOLDERID_Documents
	case "Favorites":
		knownFolderID = windows.FOLDERID_Favorites
	case "NetHood":
		knownFolderID = windows.FOLDERID_NetHood
	case "PrintHood":
		knownFolderID = windows.FOLDERID_PrintHood
	case "Templates":
		knownFolderID = windows.FOLDERID_Templates
	case "CommonStartup":
		knownFolderID = windows.FOLDERID_CommonStartup
	case "CommonPrograms":
		knownFolderID = windows.FOLDERID_CommonPrograms
	case "CommonStartMenu":
		knownFolderID = windows.FOLDERID_CommonStartMenu
	case "PublicDesktop":
		knownFolderID = windows.FOLDERID_PublicDesktop
	case "ProgramData":
		knownFolderID = windows.FOLDERID_ProgramData
	case "CommonTemplates":
		knownFolderID = windows.FOLDERID_CommonTemplates
	case "PublicDocuments":
		knownFolderID = windows.FOLDERID_PublicDocuments
	case "RoamingAppData":
		knownFolderID = windows.FOLDERID_RoamingAppData
	case "LocalAppData":
		knownFolderID = windows.FOLDERID_LocalAppData
	case "LocalAppDataLow":
		knownFolderID = windows.FOLDERID_LocalAppDataLow
	case "InternetCache":
		knownFolderID = windows.FOLDERID_InternetCache
	case "Cookies":
		knownFolderID = windows.FOLDERID_Cookies
	case "History":
		knownFolderID = windows.FOLDERID_History
	case "System":
		knownFolderID = windows.FOLDERID_System
	case "SystemX86":
		knownFolderID = windows.FOLDERID_SystemX86
	case "Windows":
		knownFolderID = windows.FOLDERID_Windows
	case "Profile":
		knownFolderID = windows.FOLDERID_Profile
	case "Pictures":
		knownFolderID = windows.FOLDERID_Pictures
	case "ProgramFilesX86":
		knownFolderID = windows.FOLDERID_ProgramFilesX86
	case "ProgramFilesCommonX86":
		knownFolderID = windows.FOLDERID_ProgramFilesCommonX86
	case "ProgramFilesX64":
		knownFolderID = windows.FOLDERID_ProgramFilesX64
	case "ProgramFilesCommonX64":
		knownFolderID = windows.FOLDERID_ProgramFilesCommonX64
	case "ProgramFiles":
		knownFolderID = windows.FOLDERID_ProgramFiles
	case "ProgramFilesCommon":
		knownFolderID = windows.FOLDERID_ProgramFilesCommon
	case "UserProgramFiles":
		knownFolderID = windows.FOLDERID_UserProgramFiles
	case "UserProgramFilesCommon":
		knownFolderID = windows.FOLDERID_UserProgramFilesCommon
	case "AdminTools":
		knownFolderID = windows.FOLDERID_AdminTools
	case "CommonAdminTools":
		knownFolderID = windows.FOLDERID_CommonAdminTools
	case "Music":
		knownFolderID = windows.FOLDERID_Music
	case "Videos":
		knownFolderID = windows.FOLDERID_Videos
	case "Ringtones":
		knownFolderID = windows.FOLDERID_Ringtones
	case "PublicPictures":
		knownFolderID = windows.FOLDERID_PublicPictures
	case "PublicMusic":
		knownFolderID = windows.FOLDERID_PublicMusic
	case "PublicVideos":
		knownFolderID = windows.FOLDERID_PublicVideos
	case "PublicRingtones":
		knownFolderID = windows.FOLDERID_PublicRingtones
	case "ResourceDir":
		knownFolderID = windows.FOLDERID_ResourceDir
	case "LocalizedResourcesDir":
		knownFolderID = windows.FOLDERID_LocalizedResourcesDir
	case "CommonOEMLinks":
		knownFolderID = windows.FOLDERID_CommonOEMLinks
	case "CDBurning":
		knownFolderID = windows.FOLDERID_CDBurning
	case "UserProfiles":
		knownFolderID = windows.FOLDERID_UserProfiles
	case "Playlists":
		knownFolderID = windows.FOLDERID_Playlists
	case "SamplePlaylists":
		knownFolderID = windows.FOLDERID_SamplePlaylists
	case "SampleMusic":
		knownFolderID = windows.FOLDERID_SampleMusic
	case "SamplePictures":
		knownFolderID = windows.FOLDERID_SamplePictures
	case "SampleVideos":
		knownFolderID = windows.FOLDERID_SampleVideos
	case "PhotoAlbums":
		knownFolderID = windows.FOLDERID_PhotoAlbums
	case "Public":
		knownFolderID = windows.FOLDERID_Public
	case "ChangeRemovePrograms":
		knownFolderID = windows.FOLDERID_ChangeRemovePrograms
	case "AppUpdates":
		knownFolderID = windows.FOLDERID_AppUpdates
	case "AddNewPrograms":
		knownFolderID = windows.FOLDERID_AddNewPrograms
	case "Downloads":
		knownFolderID = windows.FOLDERID_Downloads
	case "PublicDownloads":
		knownFolderID = windows.FOLDERID_PublicDownloads
	case "SavedSearches":
		knownFolderID = windows.FOLDERID_SavedSearches
	case "QuickLaunch":
		knownFolderID = windows.FOLDERID_QuickLaunch
	case "Contacts":
		knownFolderID = windows.FOLDERID_Contacts
	case "SidebarParts":
		knownFolderID = windows.FOLDERID_SidebarParts
	case "SidebarDefaultParts":
		knownFolderID = windows.FOLDERID_SidebarDefaultParts
	case "PublicGameTasks":
		knownFolderID = windows.FOLDERID_PublicGameTasks
	case "GameTasks":
		knownFolderID = windows.FOLDERID_GameTasks
	case "SavedGames":
		knownFolderID = windows.FOLDERID_SavedGames
	case "Games":
		knownFolderID = windows.FOLDERID_Games
	case "SEARCH_MAPI":
		knownFolderID = windows.FOLDERID_SEARCH_MAPI
	case "SEARCH_CSC":
		knownFolderID = windows.FOLDERID_SEARCH_CSC
	case "Links":
		knownFolderID = windows.FOLDERID_Links
	case "UsersFiles":
		knownFolderID = windows.FOLDERID_UsersFiles
	case "UsersLibraries":
		knownFolderID = windows.FOLDERID_UsersLibraries
	case "SearchHome":
		knownFolderID = windows.FOLDERID_SearchHome
	case "OriginalImages":
		knownFolderID = windows.FOLDERID_OriginalImages
	case "DocumentsLibrary":
		knownFolderID = windows.FOLDERID_DocumentsLibrary
	case "MusicLibrary":
		knownFolderID = windows.FOLDERID_MusicLibrary
	case "PicturesLibrary":
		knownFolderID = windows.FOLDERID_PicturesLibrary
	case "VideosLibrary":
		knownFolderID = windows.FOLDERID_VideosLibrary
	case "RecordedTVLibrary":
		knownFolderID = windows.FOLDERID_RecordedTVLibrary
	case "HomeGroup":
		knownFolderID = windows.FOLDERID_HomeGroup
	case "HomeGroupCurrentUser":
		knownFolderID = windows.FOLDERID_HomeGroupCurrentUser
	case "DeviceMetadataStore":
		knownFolderID = windows.FOLDERID_DeviceMetadataStore
	case "Libraries":
		knownFolderID = windows.FOLDERID_Libraries
	case "PublicLibraries":
		knownFolderID = windows.FOLDERID_PublicLibraries
	case "UserPinned":
		knownFolderID = windows.FOLDERID_UserPinned
	case "ImplicitAppShortcuts":
		knownFolderID = windows.FOLDERID_ImplicitAppShortcuts
	case "AccountPictures":
		knownFolderID = windows.FOLDERID_AccountPictures
	case "PublicUserTiles":
		knownFolderID = windows.FOLDERID_PublicUserTiles
	case "AppsFolder":
		knownFolderID = windows.FOLDERID_AppsFolder
	case "StartMenuAllPrograms":
		knownFolderID = windows.FOLDERID_StartMenuAllPrograms
	case "CommonStartMenuPlaces":
		knownFolderID = windows.FOLDERID_CommonStartMenuPlaces
	case "ApplicationShortcuts":
		knownFolderID = windows.FOLDERID_ApplicationShortcuts
	case "RoamingTiles":
		knownFolderID = windows.FOLDERID_RoamingTiles
	case "RoamedTileImages":
		knownFolderID = windows.FOLDERID_RoamedTileImages
	case "Screenshots":
		knownFolderID = windows.FOLDERID_Screenshots
	case "CameraRoll":
		knownFolderID = windows.FOLDERID_CameraRoll
	case "SkyDrive":
		knownFolderID = windows.FOLDERID_SkyDrive
	case "OneDrive":
		knownFolderID = windows.FOLDERID_OneDrive
	case "SkyDriveDocuments":
		knownFolderID = windows.FOLDERID_SkyDriveDocuments
	case "SkyDrivePictures":
		knownFolderID = windows.FOLDERID_SkyDrivePictures
	case "SkyDriveMusic":
		knownFolderID = windows.FOLDERID_SkyDriveMusic
	case "SkyDriveCameraRoll":
		knownFolderID = windows.FOLDERID_SkyDriveCameraRoll
	case "SearchHistory":
		knownFolderID = windows.FOLDERID_SearchHistory
	case "SearchTemplates":
		knownFolderID = windows.FOLDERID_SearchTemplates
	case "CameraRollLibrary":
		knownFolderID = windows.FOLDERID_CameraRollLibrary
	case "SavedPictures":
		knownFolderID = windows.FOLDERID_SavedPictures
	case "SavedPicturesLibrary":
		knownFolderID = windows.FOLDERID_SavedPicturesLibrary
	case "RetailDemo":
		knownFolderID = windows.FOLDERID_RetailDemo
	case "Device":
		knownFolderID = windows.FOLDERID_Device
	case "DevelopmentFiles":
		knownFolderID = windows.FOLDERID_DevelopmentFiles
	case "Objects3D":
		knownFolderID = windows.FOLDERID_Objects3D
	case "AppCaptures":
		knownFolderID = windows.FOLDERID_AppCaptures
	case "LocalDocuments":
		knownFolderID = windows.FOLDERID_LocalDocuments
	case "LocalPictures":
		knownFolderID = windows.FOLDERID_LocalPictures
	case "LocalVideos":
		knownFolderID = windows.FOLDERID_LocalVideos
	case "LocalMusic":
		knownFolderID = windows.FOLDERID_LocalMusic
	case "LocalDownloads":
		knownFolderID = windows.FOLDERID_LocalDownloads
	case "RecordedCalls":
		knownFolderID = windows.FOLDERID_RecordedCalls
	case "AllAppMods":
		knownFolderID = windows.FOLDERID_AllAppMods
	case "CurrentAppMods":
		knownFolderID = windows.FOLDERID_CurrentAppMods
	case "AppDataDesktop":
		knownFolderID = windows.FOLDERID_AppDataDesktop
	case "AppDataDocuments":
		knownFolderID = windows.FOLDERID_AppDataDocuments
	case "AppDataFavorites":
		knownFolderID = windows.FOLDERID_AppDataFavorites
	case "AppDataProgramData":
		knownFolderID = windows.FOLDERID_AppDataProgramData
	default:
		return nil, errors.New(fmt.Sprintf("Unknown FOLDERID: %#v", knownFolderID))
	}

	return knownFolderID, nil
}

func (t *SearchType) UnmarshalJSON(data []byte) error {
	var searchType string
	if err := json.Unmarshal(data, &searchType); err != nil {
		return err
	}

	switch searchType {
	case "knownFolder":
		*t = STKnownFolder
	case "registry":
		*t = STRegistry
	case "folder":
		*t = STFolder
	default:
		return errors.New(fmt.Sprintf("Invalid searchType: %s", searchType))
	}

	return nil
}

func (t *RegistryInfo) UnmarshalJSON(data []byte) error {
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if name, ok := obj["name"]; ok {
		t.Name = name
	} else {
		return errors.New(fmt.Sprintf("Invalid registry info, obj=%#v", obj))
	}

	regPath, ok := obj["path"]
	if !ok {
		return errors.New(fmt.Sprintf("Invalid registry info, obj=%#v", obj))
	}

	idx := strings.Index(regPath, "\\")
	if idx < 0 {
		return errors.New(fmt.Sprintf("Invalid registry info, obj=%#v", obj))
	}

	rootKey := regPath[:idx]
	t.Key = regPath[idx+1:]
	if len(t.Key) == 0 {
		return errors.New(fmt.Sprintf("Invalid registry info, obj=%#v", obj))
	}

	switch rootKey {
	case "HKCR", "HKEY_CLASSES_ROOT":
		t.RootKey = registry.CLASSES_ROOT
	case "HKLM", "HKEY_LOCAL_MACHINE":
		t.RootKey = registry.LOCAL_MACHINE
	case "HKCU", "HKEY_CURRENT_USER":
		t.RootKey = registry.CURRENT_USER
	case "HKU", "HKEY_USERS":
		t.RootKey = registry.USERS
	case "HKCC", "HKEY_CURRENT_CONFIG":
		t.RootKey = registry.CURRENT_CONFIG
	default:
		return errors.New(fmt.Sprintf("Invalid registry info, obj=%#v", obj))
	}
	return nil
}

func LoadGameSearchInfo(path string) *GameSearchInfo {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open %s, err=%s\n", path, err)
		return nil
	}

	defer file.Close()
	content, err := ioutil.ReadAll(file)
	searchInfo := new(GameSearchInfo)
	err = json.Unmarshal(content, searchInfo)
	if err != nil {
		log.Printf("Failed to unmarshal to json： %s\n", hex.Dump(content))
		return nil
	}

	obj := map[string]interface{}{}
	err = json.Unmarshal(content, &obj)
	if err != nil {
		log.Printf("Failed to unmarshal to json： %s\n", hex.Dump(content))
		return nil
	}

	// Parse knownFolderID, TODO find better way
	if knownFolderID, ok := obj["knownFolderID"]; ok {
		if _, ok := knownFolderID.(string); ok {
			searchInfo.FolderID, err = toKnownFolderID(knownFolderID.(string))
			if err != nil {
				log.Printf("Failed to convert string to FolderID, str=%s\n", knownFolderID.(string))
				return nil
			}
		}
	}

	if searchInfo.Reg == nil && searchInfo.FolderID == nil {
		log.Printf("Invalid game search info： %s\n", hex.Dump(content))
		return nil
	}

	return searchInfo
}

type GameInfo struct {
	Name     string
	Dir      string
	ProcName string
}

func LoadGameList(confDir string) []GameInfo {
	var gameSearchInfo []*GameSearchInfo
	err := filepath.Walk(confDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			info := LoadGameSearchInfo(path)
			if info != nil {
				gameSearchInfo = append(gameSearchInfo, info)
			}
		}

		return nil
	})

	CheckError(err)
	var gameList []GameInfo
	for _, info := range gameSearchInfo {
		if info.Name == `` || info.SubDir == `` {
			log.Printf("Invalid search info: %#v\n", info)
			continue
		}

		var dir string
		switch info.Type {
		case STKnownFolder:
			dir, err = windows.KnownFolderPath(info.FolderID, 0)
			CheckError(err)
		case STRegistry:
			key, err := registry.OpenKey(info.Reg.RootKey, info.Reg.Key, registry.QUERY_VALUE|registry.WOW64_64KEY)
			if err != nil {
				continue
			}

			dir, _, err = key.GetStringValue(info.Reg.Name)
			if err != nil {
				continue
			}
		case STFolder:
			dir = info.Dir
		default:
			log.Fatalf("Invalid search type: %d\n", info.Type)
		}

		if dir == `` {
			continue
		}

		dir = filepath.Join(dir, info.SubDir)
		valid, _ := IsDir(dir)
		if !valid {
			continue
		}

		gameList = append(gameList, GameInfo{info.Name, dir, info.ProcName})
	}

	return gameList
}
