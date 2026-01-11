import { Button } from "@/components/UI/Button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/UI/Card"
import { Input } from "@/components/UI/Input"
import { useForm } from "react-hook-form"

export default function Settings() {
    const { register, handleSubmit } = useForm({
        defaultValues: {
            apiKey: "****************",
            maxConcurrent: 5,
            proxyRotation: true
        }
    })

    const onSubmit = (data: any) => {
        console.log(data)
        alert("Settings saved!")
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">Settings</h2>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>General Configuration</CardTitle>
                    <CardDescription>Manage your scraper global settings.</CardDescription>
                </CardHeader>
                <CardContent>
                    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                        <div className="grid gap-2">
                            <label className="text-sm font-medium">Google API Key (Optional)</label>
                            <Input placeholder="Enter API Key" {...register("apiKey")} />
                        </div>
                        <div className="grid gap-2">
                            <label className="text-sm font-medium">Max Concurrent Jobs</label>
                            <Input type="number" {...register("maxConcurrent")} />
                        </div>
                        <div className="flex items-center gap-2">
                            <input type="checkbox" id="proxyRotation" {...register("proxyRotation")} className="rounded border-gray-300" />
                            <label htmlFor="proxyRotation" className="text-sm font-medium">Enable Proxy Rotation</label>
                        </div>
                        <Button type="submit">Save Changes</Button>
                    </form>
                </CardContent>
            </Card>
        </div>
    )
}
