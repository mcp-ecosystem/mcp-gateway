import { Card, CardBody, Button } from '@heroui/react';
import { t } from 'i18next';
import React, { useCallback } from 'react';
import { useDropzone } from 'react-dropzone';

import LocalIcon from '@/components/LocalIcon';
import { importOpenAPI } from '@/services/api';
import { toast } from "@/utils/toast.ts";

interface OpenAPIImportProps {
  onSuccess?: () => void;
}

const OpenAPIImport: React.FC<OpenAPIImportProps> = ({ onSuccess }) => {
  const onDrop = useCallback(async (acceptedFiles: globalThis.File[]) => {
    if (acceptedFiles.length === 0) {
      toast.error(t('errors.invalid_openapi_file'), {
        duration: 3000,
      });
      return;
    }

    try {
      await importOpenAPI(acceptedFiles[0]);
      toast.success(t('errors.import_openapi_success'), {
        duration: 3000,
      });
      onSuccess?.();
    } catch {
      toast.error(t('errors.import_openapi_failed'), {
        duration: 3000,
      })
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
          <LocalIcon icon="lucide:upload" className="text-4xl mb-4 text-primary" />
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
