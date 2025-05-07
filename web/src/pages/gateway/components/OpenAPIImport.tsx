import React from 'react';
import { useDropzone } from 'react-dropzone';
import { Card, CardBody, Button } from "@heroui/react";
import { Icon } from '@iconify/react';

interface OpenAPIImportProps {
  onSuccess: (content: string) => void;
}

const OpenAPIImport: React.FC<OpenAPIImportProps> = ({ onSuccess }) => {
  const onDrop = React.useCallback((acceptedFiles: File[]) => {
    const file = acceptedFiles[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        onSuccess(content);
      };
      reader.readAsText(file);
    }
  }, [onSuccess]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/json': ['.json'],
      'application/yaml': ['.yaml', '.yml'],
      'text/yaml': ['.yaml', '.yml']
    },
    multiple: false
  });

  return (
    <Card className="w-full">
      <CardBody>
        <div
          {...getRootProps()}
          className={`flex flex-col items-center justify-center p-6 border-2 border-dashed rounded-lg cursor-pointer transition-colors ${
            isDragActive ? 'bg-primary/10 border-primary' : 'bg-content2 border-divider'
          }`}
        >
          <input {...getInputProps()} />
          <Icon icon="lucide:upload" className="text-4xl mb-4 text-primary" />
          {isDragActive ? (
            <p className="text-lg text-primary">Drop the OpenAPI specification file here...</p>
          ) : (
            <div className="text-center">
              <p className="text-lg">Drag and drop an OpenAPI specification file here</p>
              <p className="text-sm text-default-500 mt-2">or</p>
              <Button color="primary" variant="flat" className="mt-2">
                Select a file
              </Button>
            </div>
          )}
          <p className="text-sm text-default-500 mt-4">
            Supported formats: JSON (.json), YAML (.yaml, .yml)
          </p>
        </div>
      </CardBody>
    </Card>
  );
};

export default OpenAPIImport;
