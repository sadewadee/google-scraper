import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import { Card, CardContent } from "@/components/UI/Card"
import { cn } from "@/lib/utils"

const formSchema = z.object({
    keyword: z.string().min(3, {
        message: "Keyword must be at least 3 characters.",
    }),
    location: z.string().optional(),
    depth: z.coerce.number().min(1).max(100),
    priority: z.enum(["low", "normal", "high"]),
})

export function JobForm() {
    const {
        register,
        handleSubmit,
        formState: { errors, isSubmitting },
        reset,
    } = useForm<any>({
        resolver: zodResolver(formSchema),
        defaultValues: {
            keyword: "",
            location: "",
            depth: 20,
            priority: "normal",
        },
    })

    function onSubmit(data: any) {
        // Determine wait time to simulate API call
        console.log("Submitting job:", data)
        setTimeout(() => {
            alert(JSON.stringify(data, null, 2))
            reset()
        }, 1000)
    }

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <Card>
                <CardContent className="pt-6 space-y-4">
                    <div className="space-y-2">
                        <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                            Search Keyword
                        </label>
                        <Input
                            placeholder="e.g. coffee shop near me"
                            {...register("keyword")}
                            className={cn(errors.keyword && "border-destructive")}
                        />
                        {errors.keyword && (
                            <p className="text-sm text-destructive">{String(errors.keyword.message)}</p>
                        )}
                        <p className="text-xs text-muted-foreground">The search query to send to Google Maps.</p>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                            <label className="text-sm font-medium leading-none">Location (Optional)</label>
                            <Input placeholder="e.g. Jakarta, Indonesia" {...register("location")} />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium leading-none">Max Results (Depth)</label>
                            <Input
                                type="number"
                                {...register("depth")}
                            />
                            {errors.depth && (
                                <p className="text-sm text-destructive">{String(errors.depth.message)}</p>
                            )}
                        </div>
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium leading-none">Priority</label>
                        <select
                            className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                            {...register("priority")}
                        >
                            <option value="low">Low</option>
                            <option value="normal">Normal</option>
                            <option value="high">High</option>
                        </select>
                    </div>
                </CardContent>
            </Card>

            <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => reset()}>Reset</Button>
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Creating..." : "Create Job"}
                </Button>
            </div>
        </form>
    )
}
