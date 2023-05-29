## go-insert-translated-name
`go-insert-translated-name` is a Go language version of the [insert-translated-name](https://github.com/manateelazycat/insert-translated-name) implementation that combines [DeepLX](https://github.com/OwO-Network/DeepLX) and [deno-bridge-ts](https://github.com/manateelazycat/deno-bridge-ts) for smaller memory footprint and faster translation.

### Usage
1. Clone
```
git clone https://github.com/edmondfrank/go-insert-translated-name.git
```
2. Build
```
cd go-insert-translated-name
```
```
go build
```
3. Replace the execution path inside the original deno-bridege
```diff
          ;; Start Deno process.
          (setq ,process
-               (start-process ,app-name ,process-buffer "deno" "run" "-A" "--unstable" ,ts-path ,app-name ,deno-port ,emacs-port))
-
+               (start-process ,app-name ,process-buffer "/path/to/insert-translated-name/insert-translated-name", ts-path, app-name, deno-port, emacs-port))
          ;; Make sure ANSI color render correctly.
          (set-process-sentinel
           ,process

```

4. restart emacs

Once the server is up and running, You can see the print message in the following format:

```
[/path/to/insert-translated-name.ts insert-translated-name 37395 39173]
Go bridge connected!
```



### Dependencies
This project depends on the following packages:


### License
This project is licensed under the MIT License. See the LICENSE file for more information.
