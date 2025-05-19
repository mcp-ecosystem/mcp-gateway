import { Modal, ModalProps } from "@heroui/react";
import { useEffect, useRef } from "react";

/**
 * AccessibleModal is a wrapper around HeroUI's Modal component that
 * prevents the application of aria-hidden to elements containing focus
 * to avoid accessibility issues.
 */
export function AccessibleModal(props: ModalProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!props.isOpen) return;

    // Observer to detect when aria-hidden is applied to ancestors of focused elements
    const observer = new window.MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (
          mutation.type === "attributes" &&
          mutation.attributeName === "aria-hidden" &&
          mutation.target instanceof window.Element  // Changed to Element to handle SVG elements too
        ) {
          const target = mutation.target;
          
          // Check if the element contains any element with focus
          // This includes SVG elements which might receive focus
          const activeElement = document.activeElement;
          const hasFocusedDescendant = 
            activeElement && 
            activeElement !== document.body && 
            (target.contains(activeElement) || target === activeElement);
          
          // If the element contains focus, remove aria-hidden and use inert instead if supported
          if (hasFocusedDescendant && target.getAttribute("aria-hidden") === "true") {
            // Remove the aria-hidden attribute
            target.removeAttribute("aria-hidden");
            
            // If inert attribute is supported (modern browsers), use it instead
            // as a better alternative to aria-hidden for dialog implementations
            if ('inert' in HTMLElement.prototype) {
              // Apply inert to siblings of the modal container, not to ancestors
              // that contain focused elements
              if (target.parentElement && 
                  containerRef.current && 
                  !target.contains(containerRef.current) && 
                  !containerRef.current.contains(target)) {
                target.setAttribute('inert', '');
              }
            }
          }
        }
      }
    });

    // Observe the whole document for aria-hidden changes
    observer.observe(document.body, {
      attributes: true,
      attributeFilter: ["aria-hidden", "inert"],
      subtree: true,
    });

    return () => {
      observer.disconnect();
      
      // Clean up any inert attributes we may have added
      if ('inert' in HTMLElement.prototype) {
        document.querySelectorAll('[inert]').forEach(el => {
          el.removeAttribute('inert');
        });
      }
    };
  }, [props.isOpen]);

  return (
    <div ref={containerRef} data-accessible-modal-container>
      <Modal 
        {...props} 
        scrollBehavior={props.scrollBehavior || "inside"} 
      />
    </div>
  );
}

// Re-export all Modal-related components for convenience
export {
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter
} from "@heroui/react"; 