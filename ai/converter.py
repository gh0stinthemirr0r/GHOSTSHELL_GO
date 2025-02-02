import argparse
import os
import sys
import logging
import torch
import tensorflow as tf
from tensorflow.keras.models import load_model
import h5py
import numpy as np
from abc import ABC, abstractmethod

# Configure the logger
logger = logging.getLogger('ModelConverter')
logger.setLevel(logging.DEBUG)  # Set to DEBUG for detailed logs

# Create console handler with a higher log level
ch = logging.StreamHandler()
ch.setLevel(logging.INFO)  # Set to INFO or DEBUG as needed

# Create formatter and add it to the handlers
formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
ch.setFormatter(formatter)

# Add the handlers to the logger
logger.addHandler(ch)

# Abstract Base Class for Model Converters
class ModelConverter(ABC):
    def __init__(self, input_path, output_path):
        self.input_path = input_path
        self.output_path = output_path

    @abstractmethod
    def load_model(self):
        pass

    @abstractmethod
    def convert_to_ggml(self):
        pass

    def save_ggml_model(self, ggml_model):
        """
        Saves the GGML model to the specified output path.
        """
        try:
            logger.info(f"Saving GGML model to: {self.output_path}")
            # Placeholder for actual GGML saving logic
            # Replace this with actual implementation
            with open(self.output_path, "wb") as f:
                torch.save(ggml_model, f)
            logger.info("Conversion complete.")
        except Exception as e:
            logger.error(f"Failed to save GGML model: {e}")
            raise

    def convert(self):
        """
        Full conversion process: load, convert, and save.
        """
        self.load_model()
        ggml_model = self.convert_to_ggml()
        self.save_ggml_model(ggml_model)

# PyTorch Converter
class PyTorchConverter(ModelConverter):
    def load_model(self):
        """
        Loads a PyTorch .pth model.
        """
        logger.info(f"Loading PyTorch model from: {self.input_path}")
        try:
            self.model = torch.load(self.input_path, map_location=torch.device("cpu"))
            logger.debug("PyTorch model loaded successfully.")
        except Exception as e:
            logger.error(f"Failed to load PyTorch model: {e}")
            raise

    def convert_to_ggml(self):
        """
        Converts the PyTorch model to a GGML-compatible format.
        """
        logger.info("Converting PyTorch model to GGML format...")
        try:
            ggml_model = {}
            for name, param in self.model.items():
                if isinstance(param, torch.Tensor):
                    ggml_model[name] = param.cpu().numpy()
                    logger.debug(f"Converted parameter: {name}")
            logger.debug("PyTorch model converted to GGML format successfully.")
            return ggml_model
        except Exception as e:
            logger.error(f"Failed to convert PyTorch model to GGML: {e}")
            raise

# TensorFlow Converter
class TensorFlowConverter(ModelConverter):
    def load_model(self):
        """
        Loads a TensorFlow .pb or Keras .h5 model.
        """
        logger.info(f"Loading TensorFlow model from: {self.input_path}")
        try:
            if self.input_path.endswith('.h5'):
                self.model = load_model(self.input_path)
                logger.debug("Keras model loaded successfully.")
            elif self.input_path.endswith('.pb'):
                # Placeholder for TensorFlow .pb loading logic
                # TensorFlow's SavedModel format should be used for better compatibility
                self.model = tf.saved_model.load(self.input_path)
                logger.debug("TensorFlow SavedModel loaded successfully.")
            else:
                raise ValueError("Unsupported TensorFlow model format. Use .h5 or .pb.")
        except Exception as e:
            logger.error(f"Failed to load TensorFlow model: {e}")
            raise

    def convert_to_ggml(self):
        """
        Converts the TensorFlow model to a GGML-compatible format.
        """
        logger.info("Converting TensorFlow model to GGML format...")
        try:
            ggml_model = {}
            if isinstance(self.model, tf.keras.Model):
                for layer in self.model.layers:
                    for weight in layer.weights:
                        name = weight.name
                        value = weight.numpy()
                        ggml_model[name] = value
                        logger.debug(f"Converted weight: {name}")
            else:
                # Handle other TensorFlow model types if necessary
                logger.warning("TensorFlow model type not fully supported for conversion.")
            logger.debug("TensorFlow model converted to GGML format successfully.")
            return ggml_model
        except Exception as e:
            logger.error(f"Failed to convert TensorFlow model to GGML: {e}")
            raise

# Keras Converter (if separate from TensorFlow)
class KerasConverter(ModelConverter):
    def load_model(self):
        """
        Loads a Keras .h5 model.
        """
        logger.info(f"Loading Keras model from: {self.input_path}")
        try:
            self.model = load_model(self.input_path)
            logger.debug("Keras model loaded successfully.")
        except Exception as e:
            logger.error(f"Failed to load Keras model: {e}")
            raise

    def convert_to_ggml(self):
        """
        Converts the Keras model to a GGML-compatible format.
        """
        logger.info("Converting Keras model to GGML format...")
        try:
            ggml_model = {}
            for layer in self.model.layers:
                for weight in layer.weights:
                    name = weight.name
                    value = weight.numpy()
                    ggml_model[name] = value
                    logger.debug(f"Converted weight: {name}")
            logger.debug("Keras model converted to GGML format successfully.")
            return ggml_model
        except Exception as e:
            logger.error(f"Failed to convert Keras model to GGML: {e}")
            raise

# GGML Converter Registry
class GGMLConverterRegistry:
    converters = {
        '.pth': PyTorchConverter,
        '.pt': PyTorchConverter,
        '.h5': KerasConverter,
        '.pb': TensorFlowConverter,
        # Add more mappings as needed
    }

    @classmethod
    def get_converter(cls, input_path, output_path):
        ext = os.path.splitext(input_path)[1].lower()
        converter_class = cls.converters.get(ext)
        if not converter_class:
            logger.error(f"No converter available for the file extension: {ext}")
            raise ValueError(f"No converter available for the file extension: {ext}")
        return converter_class(input_path, output_path)

def convert_model(input_path, output_path):
    """
    Determines the appropriate converter and performs the conversion.
    """
    logger.info(f"Starting conversion: {input_path} -> {output_path}")
    try:
        converter = GGMLConverterRegistry.get_converter(input_path, output_path)
        converter.convert()
    except Exception as e:
        logger.error(f"Conversion failed: {e}")
        sys.exit(1)
    logger.info("Model conversion completed successfully.")

def main():
    parser = argparse.ArgumentParser(description="Convert various AI model formats to GGML-compatible format.")
    parser.add_argument("input", type=str, help="Path to the input model file (e.g., .pth, .h5, .pb).")
    parser.add_argument("output", type=str, help="Path to save the converted GGML model (e.g., .ggml).")
    parser.add_argument("--overwrite", action="store_true", help="Overwrite the output file if it already exists.")
    parser.add_argument("--log-level", type=str, default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"],
                        help="Set the logging level.")

    args = parser.parse_args()

    # Configure logging level based on user input
    numeric_level = getattr(logging, args.log_level.upper(), None)
    if not isinstance(numeric_level, int):
        logger.error(f"Invalid log level: {args.log_level}")
        sys.exit(1)
    ch.setLevel(numeric_level)

    input_path = args.input
    output_path = args.output

    # Check if input file exists
    if not os.path.exists(input_path):
        logger.error(f"Input file '{input_path}' does not exist.")
        sys.exit(1)

    # Check if output file exists
    if os.path.exists(output_path) and not args.overwrite:
        logger.error(f"Output file '{output_path}' already exists. Use --overwrite to overwrite it.")
        sys.exit(1)

    # Perform the conversion
    convert_model(input_path, output_path)

if __name__ == "__main__":
    main()
