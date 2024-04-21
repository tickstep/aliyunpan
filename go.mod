module github.com/tickstep/aliyunpan

go 1.20

require (
	github.com/GeertJohan/go.incremental v1.0.0
	github.com/deckarep/golang-set v1.8.0
	github.com/dop251/goja v0.0.0-20220408131256-ffe77e20c6f1
	github.com/jordan-wright/email v0.0.0-20200602115436-fd8a7622303e
	github.com/json-iterator/go v1.1.12
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/oleiade/lane v0.0.0-20160817071224-3053869314bb
	github.com/olekukonko/tablewriter v0.0.2-0.20190618033246-cc27d85e17ce
	github.com/peterh/liner v1.2.1
	github.com/satori/go.uuid v1.2.0
	github.com/tickstep/aliyunpan-api v0.2.1
	github.com/tickstep/bolt v1.3.4
	github.com/tickstep/library-go v0.1.1
	github.com/urfave/cli v1.21.1-0.20190817182405-23c83030263f
)

require (
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/btcsuite/btcd v0.22.1 // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/dlclark/regexp2 v1.4.1-0.20201116162257-a2a8dda75c91 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/sys v0.0.0-20210423082822-04245dca01da // indirect
	golang.org/x/text v0.3.7 // indirect
)

//replace github.com/boltdb/bolt => github.com/tickstep/bolt v1.3.4
//replace github.com/tickstep/bolt => /Users/tickstep/Documents/Workspace/go/projects/bolt
//replace github.com/tickstep/library-go => /Users/tickstep/Documents/Workspace/go/projects/library-go
//replace github.com/tickstep/aliyunpan-api => /Users/tickstep/Documents/Workspace/go/projects/aliyunpan-api
