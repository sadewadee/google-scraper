import React, { useEffect } from "react"
import { motion, AnimatePresence } from "framer-motion"
import { X } from "lucide-react"
import { Button } from "./Button"
import { createPortal } from "react-dom"

interface DrawerProps {
    isOpen: boolean
    onClose: () => void
    title?: React.ReactNode
    children: React.ReactNode
    width?: string
}

export function Drawer({ isOpen, onClose, title, children, width = "500px" }: DrawerProps) {
    // Prevent body scroll when drawer is open
    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = "hidden"
        } else {
            document.body.style.overflow = "unset"
        }
        return () => {
            document.body.style.overflow = "unset"
        }
    }, [isOpen])

    return createPortal(
        <AnimatePresence>
            {isOpen && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                        className="fixed inset-0 bg-black/20 z-50 backdrop-blur-sm"
                    />

                    {/* Drawer Panel */}
                    <motion.div
                        initial={{ x: "100%" }}
                        animate={{ x: 0 }}
                        exit={{ x: "100%" }}
                        transition={{ type: "spring", damping: 25, stiffness: 200 }}
                        className={`fixed top-0 right-0 h-full w-full bg-neu-base shadow-2xl z-50 flex flex-col sm:max-w-[500px]`}
                        style={{ width: window.innerWidth < 640 ? "100%" : width }}
                    >
                        {/* Header */}
                        <div className="flex items-center justify-between p-4 border-b border-gray-200/50 bg-neu-base z-10">
                            <div className="font-semibold text-lg text-gray-800 truncate pr-4">
                                {title}
                            </div>
                            <Button
                                variant="icon"
                                onClick={onClose}
                                className="!p-2 hover:text-red-500 shrink-0"
                            >
                                <X className="w-5 h-5" />
                            </Button>
                        </div>

                        {/* Content */}
                        <div className="flex-1 overflow-y-auto p-4 custom-scrollbar">
                            {children}
                        </div>
                    </motion.div>
                </>
            )}
        </AnimatePresence>,
        document.body
    )
}
