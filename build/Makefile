all: clean run

dev-add.sh: ../mio/dev-add.sh
	cp $< .

dev-remove.sh: ../mio/dev-remove.sh
	cp $< .

runner-mio-bin: ../cmd/runner-mio/main.go
	go build -o ./runner-mio-bin $<

runner-node-bin: ../cmd/runner-node/main.go
	go build -o ./runner-node-bin $<

runner-bin: ../cmd/runner/main.go
	go build -o ./runner-bin $<

runner: runner-mio-bin dev-add.sh dev-remove.sh runner-node-bin runner-bin runner-config.yml
	mkdir runner
	cp $^ runner

run: runner
	sudo ./runner/runner-bin -config ./runner-config.yml

clean:
	rm -f \
		dev-add.sh \
		dev-remove.sh \
		runner-mio-bin \
		runner-node-bin \
		runner-bin
	rm -rf \
		runner

.PHONY: all run clean
