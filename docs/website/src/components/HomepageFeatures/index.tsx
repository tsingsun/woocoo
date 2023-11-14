import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: '易用性',
    Svg: require('@site/static/img/undraw_docusaurus_mountain.svg').default,
    description: (
      <>
        WooCoo 可以让你很容易就启动一个 Web 站点或 GRPC 服务, 并且可以很容易基于配置变更.
      </>
    ),
  },
  {
    title: '工程实践',
    Svg: require('@site/static/img/undraw_docusaurus_react.svg').default,
    description: (
      <>
        WooCoo 的目标是融合最佳的工程实践, 使开发者一开始就处于工程化视角, 并可以专注于代码开发.
      </>
    ),
  },
  {
    title: '高性能低成本',
    Svg: require('@site/static/img/intro_performance.svg').default,
    description: (
      <>
        基于优秀的Golang语言, 有着优秀的性能表现及极低的成本需求, 极具性价比.
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
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

export default function HomepageFeatures() {
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
