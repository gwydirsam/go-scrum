install::
	cd cmd/scrum; govvv install

dev::
	cd cmd/scrum; govvv build -o ../../scrum

get-tools::
	go get -u github.com/ahmetb/govvv
