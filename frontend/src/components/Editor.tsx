import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useState } from "react";
import axios from "axios";
import StarterKit from "@tiptap/starter-kit";
import { safe } from "../utils.ts";
import { DocumentSchema, type DocumentType } from "../types.ts";

type SaveStatus = "Idle" | "Saving..." | "Saved!" | "Error";

const TiptapEditor = () => {
    const [saveStatus, setSaveStatus] = useState<SaveStatus>("Idle");

    const editor = useEditor({
        extensions: [StarterKit],
        content: `<p>Loading content...</p>`,
        editorProps: {
            attributes: {
                class: "prose dark:prose-invert prose bg-gray-400\
                sm:prose-base lg:prose-lg xl:prose-2xl m-5 focus:outline-none text-white",
            },
        },
    });

    const documentID = "d1c4e122-85e5-4887-84b8-3afbba53570a";
    useEffect(() => {
        if (!editor) {
            return;
        }

        const fetchDocument = async () => {
            const [response, error] = await safe(
                axios.get(`http://localhost:8080/documents/${documentID}`),
            );

            const parsedResponse: DocumentType = DocumentSchema.parse(
                response?.data,
            );
            if (!parsedResponse) {
                return;
            }

            if (error || !parsedResponse?.content) {
                console.error(
                    "Failed to fetch document:",
                    error?.message ?? "No data",
                );
                editor.commands.setContent("Failed to get the content");
                return;
            }

            editor.commands.setContent(parsedResponse.content);
        };
        fetchDocument();
    }, [editor]);

    const handleSave = async () => {
        if (!editor) {
            return;
        }

        setSaveStatus("Saving...");

        const htmlContent = editor.getHTML();
        const [response, error] = await safe(
            axios.put(`http://localhost:8080/documents/${documentID}`, {
                title: "My Updated Title",
                content: htmlContent,
            }),
        );

        const parsedResponse: DocumentType = DocumentSchema.parse(
            response?.data,
        );

        if (error || !parsedResponse.content) {
            console.error(
                "Failed to fetch document:",
                error?.message ?? "No data",
            );
            console.error("Failed to update document");
            return;
        }

        setSaveStatus("Saved!");

        console.info("docID: ", parsedResponse.id);
        setTimeout(() => setSaveStatus("Idle"), 2000);
    };

    return (
        <div className="border border-gray-300 dark:border-gray-700 rounded-lg p-2 bg-white dark:bg-gray-800 shadow-md">
            <div className="flex justify-end mb-2">
                <button
                    type="submit"
                    onClick={handleSave}
                    className="px-4 py-2 text-sm font-semibold text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
                    disabled={saveStatus === "Saving..."}
                >
                    {saveStatus}
                </button>
            </div>
            <EditorContent editor={editor} />
        </div>
    );
};

export default TiptapEditor;
