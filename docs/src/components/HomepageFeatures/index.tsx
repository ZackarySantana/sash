import type { ReactNode } from "react";
import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

type FeatureItem = {
    title: string;
    Svg: React.ComponentType<React.ComponentProps<"svg">>;
    description: ReactNode;
};

const FeatureList: FeatureItem[] = [
    {
        title: "One config, three commands",
        Svg: require("@site/static/img/undraw_docusaurus_mountain.svg").default,
        description: (
            <>
                <strong>sash.json</strong> holds your frontend scripts, so{" "}
                <strong>sash dev</strong>, <strong>sash build</strong>, and{" "}
                <strong>sash bind</strong> stay aligned without copy-pasted
                shell snippets. Fewer moving parts means fewer “works on my
                machine” rituals.
            </>
        ),
    },
    {
        title: "RPC from plain Go methods",
        Svg: require("@site/static/img/undraw_docusaurus_tree.svg").default,
        description: (
            <>
                Write methods on a struct; <strong>sash bind</strong> turns them
                into typed TypeScript calls and Go mount helpers. You spend time
                on behavior, not hand-maintained URLs and JSON envelopes.
            </>
        ),
    },
    {
        title: "Embedded UI, or split dev when you need HMR",
        Svg: require("@site/static/img/undraw_docusaurus_react.svg").default,
        description: (
            <>
                Ship assets inside the binary for a single-loopback story, or
                let <strong>sash dev</strong> open Vite while Go listens on its
                own port. Bindings notice <strong>Vite dev</strong> vs{" "}
                <strong>embedded prod</strong> so you rarely configure{" "}
                <code>apiBase</code> by hand.
            </>
        ),
    },
];

function Feature({ title, Svg, description }: FeatureItem) {
    return (
        <div className={clsx("col col--4")}>
            <div className="text--center">
                <Svg className={styles.featureSvg} role="img" />
            </div>
            <div className="text--center padding-horiz--md">
                <Heading as="h3">{title}</Heading>
                <p>{description}</p>
            </div>
        </div>
    );
}

export default function HomepageFeatures(): ReactNode {
    return (
        <section className={styles.features}>
            <div className="container">
                <div className="row">
                    {FeatureList.map((props, idx) => (
                        <Feature key={idx} {...props} />
                    ))}
                </div>
            </div>
        </section>
    );
}
