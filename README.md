# Application migration for SBDIOI40

Package sbdioi40 provides functionalities for the migration of applications between
OpenStack platforms belonging to the SBDIOI40 project.

## Conventions

The package is build around the concept of **application**: an application in SBDIOI40 is a set of virtual resources that provides functionalities for one partner of the project. An application is hosted within an OpenStack platform, and it consists of:

- 1 virtual network
- N virtual machines
- N ports, each of which connects a V.M. to the network

Some conventions must be respected when configuring a new application. Given an application called "app" with services "one", "two" and "three":

- the virtual network must be called "appnet"
- the virtual machines "apponevm", "apptwovm" and "appthreevm"
- the ports "apponeport", "apptwoport" and "appthreeport"

Also, all applications must live in a project called "sbdioi40".

If all these preconditions are met, the package will be able to identify applications and manage them successfully.

## Basic usage

First, you should connect to a platform. The package exposes a function `Connect` that lets you connect to an OpenStack platorm belonging to the SBDIOI40 project. The function takes as input the login information and returns an object of type `Platform` which represents an established connection to a platform.

```go
plat, err := sbdioi40.Connect("addr", "user", "password")
if err != nil {
    log.Fatal(err)
}
```

`Platform` objects are central to the usage of the library because it exposes methods with which you can manage your applications.

The method `ListApplications` lets you retrieve information about all the applications that are currently hosted on the given platform. `Application` retrieves information about one specific application, given its name.

## Migration

The main feature of the package is that it lets you migrate an application between distinct platforms.

With the method `Snapshot` you can take a snapshot of an application and download it to the local storage. The result is a `Snapshot` object which represent the picture that you have taken at a certain moment in time.

```go
snap, err := srcPlat.Snapshot("appname")
if err != nil {
    log.Fatal(err)
}
```

Once a snapshot has been taken, you can implement the migration by restoring it to a different platform. To restore a snapshot, the platform object exposes a method `Restore` that takes a snapshot as input and rebuilds the application in the given platform.

```go
if err := dstPlat.Restore(snap); err != nil {
    log.Fatal(err)
}
```

As a result, the application has successfully moved between two different platforms.

## Further information

For more details, you can refer to the deliverable for the SBDIOI40 project. (link coming soon)
