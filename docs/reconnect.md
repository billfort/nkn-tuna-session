# Tuna Session Connection Disconnect and Reconnect

## Background

Tuna session is a session control protocol that uses NCP session over Tuna connections. A Tuna session has two session endpoints: Listener and Dialer. There is a configuration parameter, `NumTunaListeners`, to configure the number of connections that will be set up when creating a Tuna session.

At the Tuna session listener side, it starts `NumTunaListeners` listeners to accept connections and creates an NCP session when a new connection arrives with the meta-data that contains the session ID. As a result, a Tuna session listener can accept and create multiple NCP sessions.

At the Tuna session dialer, it queries the other endpoint using NKN multi-client to get the public address of the Tuna node to connect to. It creates the same number of connections.

Once the dialer and listener's sessions are set up, they can begin writing and reading data over these multi-connections.

Since NKN network nodes are deployed all over the world, they are subjected to varying circumstances. When a Tuna session is working, some connections may be disconnected unexpectedly. Although the session can still work even without full connections, it is still better to set up a new connection if one connection disconnects unexpectedly.

The following tasks are performed in this process:

- Tracking the connection status and determining the difference between normal and abnormal connection closures.
- Setting up a new connection if a previous connection disconnects unexpectedly.
- Maintaining consistency in data and session work.

## How to Determine If a Connection is Disconnected Exceptionally

In the Tuna Session, connections are used to write and read messages. When reading or writing messages, an error is returned if the connection is closed or if exceptions occur, such as a timeout or network error.

To determine if a connection is closed normally by the application or disconnected exceptionally, the following logic can be used:
If a connection is disconnected exceptionally, the session should still be open. When attempting to read or write on a disconnected connection, an `IO.EOF` error should be returned:

```
if (!session.IsClosed() && err == IO.EOF) {
    // The connection is disconnected exceptionally
    // Create a new connection
}
```

## Creating a New Connection

- Tuna Session Listener
  A Tuna Session Listener starts a `tunaExit`, and `tunaExit` starts `StartReverse(true)`.
  When a new connection is needed at the Tuna Session Listener side, a new `tunaExit` must be created.

```
newTunaExit(i int)
```

In the Tuna Session Listener, an infinite loop is started in the `listenNet(i int)` function. When a new connection arrives, it is necessary to replace the old disconnected connection in the loop.

```
c.sessionConns[sessKey][connID(i)] = conn
```

- Tuna Session Dialer
  Before the Tuna Session Dialer starts a new connection, it must inquire about the new public addresses from the session listener.

```
buf, err := json.Marshal(&Request{Action: "getPubAddr"})
if err != nil {
    return nil, err
}

respChan, err := c.multiClient.Send(nkn.NewStringArray(remoteAddr), buf, nil)
if err != nil {
    return nil, err
}

```

After obtaining the new public address, we must compare it to the current normal connection's public address and get the public address that is needed to create the new connection.

At the Tuna Session Dialer side, a new dialer connection is started based on the disconnected connection's public address.

```
netConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", pubAddrs.Addrs[i].IP, pubAddrs.Addrs[i].Port))
// replace old disconnected conn
```

If an NCP session has only one connection, it is usually closed when that connection is closed. If a new connection is created to replace the old one, a new NCP session must also be established.

## Keep session status consistance

A Tuna Session has two endpoints: Listener and Dialer. Each endpoint has the following data structures to maintain the session's status:

```
	listeners        []net.Listener
	tunaExits        []*tuna.TunaExit
	sessions         map[string]*ncp.Session
	sessionConns     map[string]map[string]*Conn
	connCount        map[string]int
	closedSessionKey *gocache.Cache
	isClosed         bool
```

When a connection is disconnected exceptionally, it is necessary to consistently update the relevant data structures. After a new connection has been established, these data structures should be updated again to ensure their consistency.
