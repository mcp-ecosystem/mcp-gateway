import { Card, CardBody, Button } from '@heroui/react';
import { Icon } from '@iconify/react';
import React, { useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { toast } from 'react-hot-toast';

interface OpenAPIImportProps {
  onSuccess?: () => void;
}

const OpenAPIImport: React.FC<OpenAPIImportProps> = ({ onSuccess }) => {
  const onDrop = useCallback(async (acceptedFiles: globalThis.File[]) => {
    if (acceptedFiles.length === 0) {
      toast.error('Please select a valid OpenAPI specification file', {
        duration: 3000,
        position: 'bottom-right',
      });
      return;
    }

    const file = acceptedFiles[0];
    const formData = new globalThis.FormData();
    formData.append('file', file);

    try {
      const response = await fetch('/api/openapi/import', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to import OpenAPI specification');
      }

      toast.success('Successfully imported OpenAPI specification', {
        duration: 3000,
        position: 'bottom-right',
      });
      onSuccess?.();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to import OpenAPI specification', {
        duration: 3000,
        position: 'bottom-right',
      });
    }
  }, [onSuccess]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/json': ['.json'],
      'application/yaml': ['.yaml', '.yml'],
    },
    multiple: false,
  });

  return (
    <Card className="w-full">
      <CardBody>
        <div
          {...getRootProps()}
          className={`flex flex-col items-center justify-center p-6 border-2 border-dashed rounded-lg cursor-pointer transition-colors ${
            isDragActive ? 'bg-content2' : 'bg-background'
          }`}
        >
          <input {...getInputProps()} />
          <Icon icon="lucide:upload" className="text-4xl mb-4" />
          {isDragActive ? (
            <p className="text-lg">Drop the OpenAPI specification file here...</p>
          ) : (
            <div className="text-center">
              <p className="text-lg">Drag and drop an OpenAPI specification file here</p>
              <p className="text-sm text-gray-500 mt-2">or</p>
              <Button color="primary" variant="flat" className="mt-2">
                Select a file
              </Button>
            </div>
          )}
          <p className="text-sm text-gray-500 mt-4">
            Supported formats: JSON (.json), YAML (.yaml, .yml)
          </p>
        </div>
      </CardBody>
    </Card>
  );
};

export default OpenAPIImport;
