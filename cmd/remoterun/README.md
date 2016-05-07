*WARNING: Use this program only on a safe-network. This is designed to be
easy to setup and use. However, it is also easy to hack.*

`remoterun` allows one computer to listen to a port and receive requests to run
a program. This can be nicely combined with `watchrun` to start programs
on a remote computer automatically when your code changes.

To set it up, on a remote computer run `remoterun`. This will start the
running server.

To force the computer to start something run
`remoterun -addr "OTHER:8080" -send program.exe`,
assuming that `OTHER` is the server computers name and it is on the same network.

To combine this with `watchrun`, do: `watchrun go build . ;; remoterun -addr "OTHER:8080" -send program.exe`.