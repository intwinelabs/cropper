all: install-deps dlib models test ## Make all

clean: ## Cleans all build artifacts
	@rm -rf dlib*
	@rm -rf models
	@rm -rf tests
	@rm -f goface_tests.tar.gz

dlib: ## Builds dlib v19.16
	@wget http://dlib.net/files/dlib-19.16.tar.bz2
	@tar xvf dlib-19.16.tar.bz2
	@cd dlib-19.16/ && mkdir build && cd build && cmake .. && cmake --build . --config Release && sudo make install && sudo ldconfig
	
install-deps: ## Installs: libjpeg-turbo8-dev, build-essential, and cmake
	@sudo apt install libjpeg-turbo8-dev build-essential cmake

models: ## Downloads models from our public blob storage
	@mkdir models
	@cd models && wget https://intwinepublic.blob.core.windows.net/ai-train-data/dlib_face_recognition_resnet_model_v1.dat.bz2
	@cd models && bunzip2 dlib_face_recognition_resnet_model_v1.dat.bz2
	@cd models && wget https://intwinepublic.blob.core.windows.net/ai-train-data/shape_predictor_5_face_landmarks.dat.bz2
	@cd models && bunzip2 shape_predictor_5_face_landmarks.dat.bz2 

test:  ## Runs tests on the library, this is slow because of the C++ build time
	@wget https://intwinepublic.blob.core.windows.net/ai-train-data/cropper_tests.tar.gz
	@tar xvzf cropper_tests.tar.gz
	@go test -v

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
