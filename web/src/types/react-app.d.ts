<<<<<<< VITE_runtime_varables
/// <reference types="react" />
/// <reference types="react-dom" />

declare namespace JSX {
  interface IntrinsicElements {
    [elemName: string]: any;
=======
import * as React from 'react';

declare namespace JSX {
  interface IntrinsicElements {
    [elemName: string]: React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
>>>>>>> main
  }
}
